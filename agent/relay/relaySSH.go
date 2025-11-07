package relay

import (
	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/agent/ssh/transport/http"
	"Goauld/common/log"
	"context"
	"errors"
	"io"
	"net"
	"strings"
)

func ListenSSHRelay(mode string, ctx context.Context, dnsTransport *transport.DNSSH) error {
	listener, err := net.ListenTCP("tcp", nil)
	if err != nil {
		return err
	}

	addr, ok := listener.Addr().(*net.TCPAddr)
	if !ok {
		return errors.New("invalid addr")
	}
	log.Info().Str("Mode", "Relay").Int("Port", addr.Port).Msg("SSH Relay listening")

	for {
		agentConn, err := listener.Accept()
		if err != nil {
			log.Debug().Err(err).Msg("SSH Relay Accept error")

			continue
		}
		rawID := make([]byte, 32)
		n, err := agentConn.Read(rawID)
		if err != nil {
			log.Error().Err(err).Msg("Error reading agent ID")
			agentConn.Close()

			continue
		}
		id := string(rawID[:n])
		serverConn, err := GetSSHCon(mode, id, ctx, dnsTransport)
		if err != nil {
			log.Debug().Err(err).Str("Mode", mode).Msg("Error relaying SSH connection")

			continue
		}
		go relay(agentConn, serverConn)
	}
}

// relay connects agent <-> server and ensures cleanup.
func relay(agentConn, serverConn net.Conn) {
	defer agentConn.Close()
	defer serverConn.Close()

	done := make(chan struct{}, 2)

	go func() {
		_, _ = io.Copy(serverConn, agentConn)
		done <- struct{}{}
	}()
	go func() {
		_, _ = io.Copy(agentConn, serverConn)
		done <- struct{}{}
	}()

	// Wait for one side to close, then clean up
	<-done
}

type result struct {
	conn net.Conn
	err  error
}

func GetSSHCon(mode string, id string, ctx context.Context, dnsTransport *transport.DNSSH) (net.Conn, error) {
	log.Info().Str("Mode", mode).Dur("Timeout", config.Get().GetSSHTimeout()).Msg("Connecting to ssh")
	timeoutCtx, cancel := context.WithTimeout(ctx, config.Get().GetSSHTimeout())
	resultChan := make(chan result, 1)

	go func() {
		var res result
		switch {
		case strings.HasPrefix(mode, "ssh"):
			dialer := net.Dialer{}
			res.conn, res.err = dialer.DialContext(timeoutCtx, "tcp", config.Get().ControlSSHServer())

		case strings.HasPrefix(mode, "quic"):
			res.conn, res.err = transport.GetQuicConn(timeoutCtx, id)

		case strings.HasPrefix(mode, "tls"):
			res.conn, res.err = transport.GetTLSConn(timeoutCtx, id)

		case strings.HasPrefix(mode, "ws"):
			res.conn, res.err = transport.GetWebsocketConn(timeoutCtx, ctx, id)

		case strings.HasPrefix(mode, "http"):
			var stream *http.SSHTTP
			stream, res.err = http.NewSSHTTP(config.Get().SSHTTPURL(id))
			res.conn = stream.Stream

		case strings.HasPrefix(mode, "dns"):
			if dnsTransport != nil {
				res.conn, res.err = dnsTransport.Session.OpenStream()
			} else {
				res.err = errors.New("DNS transport is unavailable")
			}
		}
		resultChan <- res
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			log.Error().Err(res.err).Str("Mode", mode).Msg("Failed to relay to ssh server")
			cancel()
		}
		if res.conn != nil {
			return res.conn, res.err
		} else {
			cancel()
		}
	case <-timeoutCtx.Done():
		log.Warn().Str("Mode", mode).Msg("Connection timed out, trying next...")
		cancel()

		return nil, timeoutCtx.Err()
	}

	return nil, errors.New("failed to relay SSH connection")
}
