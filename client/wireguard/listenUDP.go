package wireguard

import (
	"Goauld/common/log"
	"Goauld/common/wireguard/udptunnel"
	"context"
	"encoding/gob"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jpillora/chisel/share/cio"
	"github.com/jpillora/chisel/share/settings"
	"golang.org/x/crypto/ssh"
	"golang.org/x/sync/errgroup"
)

// ListenUDP is a special listener which forwards packets via
// the bound ssh connection. tricky part is multiplexing lots of
// udp clients through the entry node. each will listen on its
// own source-port for a response:
//
//	                                            (random)
//	src-1 1111->...                         dst-1 6345->7777
//	src-2 2222->... <---> udp <---> udp <-> dst-1 7543->7777
//	src-3 3333->...    listener    handler  dst-1 1444->7777
//
// we must store these mappings (1111-6345, etc) in memory for a length
// of time, so that when the exit node receives a response on 6345, it
// knows to return it to 1111.
func ListenUDP(tcpConn net.Conn) (*UdpListener, error) {
	a, err := net.ResolveUDPAddr("udp", "127.0.0.1:33333")
	if err != nil {
		return nil, fmt.Errorf("resolve: %w", err)
	}
	conn, err := net.ListenUDP("udp", a)
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	// ready
	u := &UdpListener{
		tcpConn: tcpConn,
		//remote:  remote,
		inbound: conn,
		maxMTU:  settings.EnvInt("UDP_MAX_SIZE", 9012),
	}
	log.Trace().Int("MTU", u.maxMTU).Msg("ListenUDP MTU")

	return u, nil
}

type UdpListener struct {
	*cio.Logger

	tcpConn net.Conn
	// remote      *settings.Remote
	inbound     *net.UDPConn
	outboundMut sync.Mutex
	outbound    *udptunnel.UdpChannel
	sent, recv  int64
	maxMTU      int
}

func (u *UdpListener) Run(ctx context.Context) error {
	defer u.inbound.Close()
	// udp doesnt accept connections,
	//udp simply forwards packets
	//and therefore only needs to listen
	eg, ctx := errgroup.WithContext(ctx)
	eg.Go(func() error {
		return u.runInbound(ctx)
	})
	eg.Go(func() error {
		return u.runOutbound(ctx)
	})
	if err := eg.Wait(); err != nil {
		log.Debug().Err(err).Msg("udp listen error")
		// u.Debugf("listen: %s", err)
		return err
	}
	log.Debug().Msg("udp listen done")
	// u.Debugf("Close (sent %s received %s)", sizestr.ToString(u.sent), sizestr.ToString(u.recv))
	return nil
}

func (u *UdpListener) runInbound(ctx context.Context) error {
	buff := make([]byte, u.maxMTU)
	for !isDone(ctx) {
		// read from inbound udp
		u.inbound.SetReadDeadline(time.Now().Add(time.Second))
		n, addr, err := u.inbound.ReadFromUDP(buff)
		if e, ok := err.(net.Error); ok && (e.Timeout() || e.Temporary()) {
			continue
		}
		if err != nil {
			return fmt.Errorf("read error: %w", err)
		}
		// upsert ssh channel
		uc, err := u.getUDPChan(ctx)
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}

			return fmt.Errorf("inbound-udpchan: %w", err)
		}
		// send over channel, including source address
		b := buff[:n]
		if err := uc.Encode(addr.String(), b); err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue // dropped packet...
			}

			return fmt.Errorf("encode error: %w", err)
		}
		// stats
		atomic.AddInt64(&u.sent, int64(n))
	}

	return nil
}

func (u *UdpListener) runOutbound(ctx context.Context) error {
	for !isDone(ctx) {
		// upsert ssh channel
		uc, err := u.getUDPChan(ctx)
		if err != nil {
			if strings.HasSuffix(err.Error(), "EOF") {
				continue
			}

			return fmt.Errorf("outbound-udpchan: %w", err)
		}
		// receive from channel, including source address
		p := udptunnel.UdpPacket{}
		if err := uc.Decode(&p); errors.Is(err, io.EOF) {
			// outbound ssh disconnected, get new connection...
			continue
		} else if err != nil {
			return fmt.Errorf("decode error: %w", err)
		}
		// write back to inbound udp
		addr, err := net.ResolveUDPAddr("udp", p.Src)
		if err != nil {
			return fmt.Errorf("resolve error: %w", err)
		}
		n, err := u.inbound.WriteToUDP(p.Payload, addr)
		if err != nil {
			return fmt.Errorf("write error: %w", err)
		}
		// stats
		atomic.AddInt64(&u.recv, int64(n))
	}

	return nil
}

func (u *UdpListener) getUDPChan(ctx context.Context) (*udptunnel.UdpChannel, error) {
	u.outboundMut.Lock()
	defer u.outboundMut.Unlock()
	// cached
	if u.outbound != nil {
		return u.outbound, nil
	}
	// ready
	o := &udptunnel.UdpChannel{
		R: gob.NewDecoder(u.tcpConn),
		W: gob.NewEncoder(u.tcpConn),
		C: u.tcpConn,
	}
	u.outbound = o
	log.Debug().Msgf("UDP channel created")

	return o, nil
}

func (u *UdpListener) unsetUDPChan(sshConn ssh.Conn) {
	sshConn.Wait()
	log.Debug().Msgf("UDP channel closed")
	u.outboundMut.Lock()
	u.outbound = nil
	u.outboundMut.Unlock()
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
