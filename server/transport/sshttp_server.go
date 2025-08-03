package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
	"www.bamsoftware.com/git/dnstt.git/turbotunnel"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"www.bamsoftware.com/git/champa.git/encapsulation"
)

const maxPayloadLength = 64 * 1024

type SSHHttpServer struct {
	pconn   *turbotunnel.QueuePacketConn
	kcpConn *kcp.Listener
	db      *persistence.DB
	store   *store.AgentStore
}

// handleStream bidirectionally connects a client stream with a TCP socket
// addressed by upstream.
func (s *SSHHttpServer) handleStream(stream *smux.Stream, upstream string, conv uint32) error {
	dialer := net.Dialer{
		Timeout: upstreamDialTimeout,
	}
	upstreamConn, err := dialer.Dial("tcp", upstream)
	if err != nil {
		return fmt.Errorf("stream %08x:%d connect upstream: %v", conv, stream.ID(), err)
	}
	//nolint:errcheck
	defer upstreamConn.Close()
	upstreamTCPConn := upstreamConn.(*net.TCPConn)

	// The client first sends its ID before transferring the conn to the SSH client
	// The ID is a MD5 hash
	rawId := make([]byte, 32)
	n, err := stream.Read(rawId)
	if err != nil {
		return fmt.Errorf("stream %08x:%d read ID fail", conv, stream.ID())
	}
	id := string(rawId[:n])

	err = s.db.SetAgentSshMode(id, "HTTP", stream.RemoteAddr().String())
	if err != nil {
		log.Warn().Err(err).Str("Agent.Id", id).Str("Mode", "HTTP").Msg("error setting agent connection mode")
	}
	s.store.SshttpAddAgent(upstreamConn, stream, id)

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := io.Copy(stream, upstreamTCPConn)
		if err == io.EOF {
			// smux Stream.Write may return io.EOF.
			err = nil
		}
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			log.Debug().Str("Mode", "SSHTTP").Msgf("stream %08x:%d copy stream←upstream: %v", conv, stream.ID(), err)
		}
		_ = upstreamTCPConn.CloseRead()
		_ = stream.Close()
	}()
	go func() {
		defer wg.Done()
		_, err := io.Copy(upstreamTCPConn, stream)
		if err == io.EOF {
			// smux Stream.WriteTo may return io.EOF.
			err = nil
		}
		if err != nil && !errors.Is(err, io.ErrClosedPipe) {
			log.Debug().Str("Mode", "SSHTTP").Msgf("stream %08x:%d copy upstream←stream: %v", conv, stream.ID(), err)
		}
		_ = upstreamTCPConn.CloseWrite()
	}()
	wg.Wait()

	return nil
}

// acceptStreams wraps a KCP session in a smux.Session,
// then awaits smux streams. It passes each stream to handleStream.
func (s *SSHHttpServer) acceptStreams(conn *kcp.UDPSession, upstream string) error {
	// Put an smux session on top of the KCP connection.
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 2
	smuxConfig.KeepAliveTimeout = idleTimeout
	smuxConfig.MaxReceiveBuffer = 16 * 1024 * 1024 // default is 4 * 1024 * 1024
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024   // default is 65 536
	sess, err := smux.Server(conn, smuxConfig)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer sess.Close()

	for {
		stream, err := sess.AcceptStream()
		if err != nil {
			//nolint:staticcheck // SA1019
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			if errors.Is(err, io.ErrClosedPipe) {
				// We don't want to report this error.
				err = nil
			}
			return err
		}
		log.Debug().Str("Mode", "SSHTTP").Msgf("begin stream %08x:%d", conn.GetConv(), stream.ID())
		go func() {
			defer func() {
				log.Debug().Str("Mode", "SSHTTP").Msgf("end stream %08x:%d", conn.GetConv(), stream.ID())
				_ = stream.Close()
			}()
			err := s.handleStream(stream, upstream, conn.GetConv())
			if err != nil {
				log.Debug().Str("Mode", "SSHTTP").Msgf("stream %08x:%d handleStream: %v", conn.GetConv(), stream.ID(), err)
			}
		}()
	}
}

// acceptSessions listens for incoming KCP connections and passes them to
// acceptStreams.
func (s *SSHHttpServer) acceptSessions(ln *kcp.Listener, upstream string) error {
	for {
		conn, err := ln.AcceptKCP()
		if err != nil {
			//nolint:staticcheck // SA1019
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			return err
		}
		log.Debug().Str("Mode", "SSHTTP").Msgf("begin session %08x", conn.GetConv())
		// Permit coalescing the payloads of consecutive sending.
		//nolint:staticcheck // SA1019
		conn.SetStreamMode(true)
		// Disable the dynamic congestion window (limit only by the
		// maximum of local and remote static windows).
		conn.SetNoDelay(
			0, // default nodelay
			0, // default interval
			0, // default resend
			1, // nc=1 => congestion window off
		)
		conn.SetWindowSize(1024, 1024) // Default is 32, 32.
		go func() {
			defer func() {
				log.Debug().Str("Mode", "SSHTTP").Msgf("end session %08x", conn.GetConv())
				_ = conn.Close()
			}()
			err := s.acceptStreams(conn, upstream)
			if err != nil && !errors.Is(err, io.ErrClosedPipe) {
				log.Debug().Str("Mode", "SSHTTP").Msgf("session %08x acceptStreams: %v", conn.GetConv(), err)
			}
		}()
	}
}

// decodeRequest extracts a ClientID and a payload from an incoming HTTP
// request. In case of a decoding failure, the returned payload slice will be
// nil. The payload is always non-nil after a successful decoding, even if the
// payload is empty.
func decodeRequest(req *http.Request) (turbotunnel.ClientID, []byte) {
	// Check the version indicator of the incoming client–server protocol.
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return turbotunnel.ClientID{}, nil
	}
	//nolint:errcheck
	defer req.Body.Close()

	var clientID turbotunnel.ClientID
	n := copy(clientID[:], body)
	if n != len(clientID) {
		return turbotunnel.ClientID{}, nil
	}
	payload := body[n:]
	return clientID, payload
}

func (s *SSHHttpServer) ServeHTTP(rw http.ResponseWriter, req *http.Request) {

	if req.Method == http.MethodGet {
		http.Error(rw, "OK", http.StatusOK)
		return
	}
	if req.Method != http.MethodPost {
		http.Error(rw, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
		return
	}

	rw.WriteHeader(http.StatusOK)

	clientID, payload := decodeRequest(req)
	if payload == nil {
		// Could not decode the client request. We do not even have a
		// meaningful clientID or nonce. This may be a result of the
		// client deliberately sending a short request for traffic
		// shaping purposes. Send back a dummy, though still
		// AMP-compatible, response.
		// TODO: random padding.
		return
	}

	// Read incoming packets from the payload.
	r := bytes.NewReader(payload)
	for {
		p, err := encapsulation.ReadData(r)
		if err != nil {
			break
		}
		s.pconn.QueueIncoming(p, clientID)
	}

	limit := maxPayloadLength
	// We loop and bundle as many outgoing packets as will fit, up to
	// maxPayloadLength. We wait up to maxResponseDelay for the first
	// available packet; after that we only include whatever packets are
	// immediately available.
	timer := time.NewTimer(maxResponseDelay)
	defer timer.Stop()
	first := true
	for {
		var p []byte
		unstash := s.pconn.Unstash(clientID)
		outgoing := s.pconn.OutgoingQueue(clientID)
		// Prioritize taking a packet first from the stash, then from
		// the outgoing queue, then finally check for expiration of the
		// timer. (We continue to bundle packets even after the timer
		// expires, as long as the packets are immediately available.)
		select {
		case p = <-unstash:
		default:
			select {
			case p = <-unstash:
			case p = <-outgoing:
			default:
				select {
				case p = <-unstash:
				case p = <-outgoing:
				case <-timer.C:
				}
			}
		}
		// We wait for the first packet only. Later packets must be
		// immediately available.
		timer.Reset(0)

		if len(p) == 0 {
			// Timer expired, we are done bundling packets into this
			// response.
			break
		}

		limit -= len(p)
		if !first && limit < 0 {
			// This packet doesn't fit in the payload size limit.
			// Stash it so that it will be first in line for the
			// next response.
			s.pconn.Stash(p, clientID)
			break
		}
		first = false

		// Write the packet to the AMP response.
		_, err := encapsulation.WriteData(rw, p)
		if err != nil {
			log.Get().Debug().Msgf("encapsulation.WriteData: %v", err)
			break
		}
		if rw, ok := rw.(http.Flusher); ok {
			rw.Flush()
		}
	}
}

func NewSSHHttpServer(store *store.AgentStore, db *persistence.DB) (*SSHHttpServer, error) {

	// noiseConn is the packet interface that communicates with the AMP/HTTP
	// Handler; it deals in encrypted Noise messages. plainConn is the
	// packet interface that communicates with KCP. noiseLoop sits in the
	// middle, handling Noise handshakes and sessions, and
	// encrypting/decrypting between the two net.PacketConns.
	plainConn := turbotunnel.NewQueuePacketConn(turbotunnel.DummyAddr{}, idleTimeout*2)

	ln, err := kcp.ServeConn(nil, 0, 0, plainConn)
	if err != nil {
		return nil, fmt.Errorf("opening KCP listener: %v", err)
	}

	server := &SSHHttpServer{
		pconn:   plainConn,
		kcpConn: ln,
		db:      db,
		store:   store,
	}

	go func() {
		err := server.acceptSessions(ln, config.Get().LocalSShAddr())
		log.Info().Str("Mode", "SSHTTP").Err(err).Msg("ssh http server accept sessions")
	}()

	return server, nil
}

func (s *SSHHttpServer) Close() error {
	return errors.Join(
		s.pconn.Close(),
		s.kcpConn.Close(),
	)
}
