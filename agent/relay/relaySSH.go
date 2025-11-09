package relay

import (
	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/agent/ssh/transport/http"
	"Goauld/common/log"
	commonnet "Goauld/common/net"
	"context"
	"errors"
	"io"
	"net"
	netHttp "net/http"
	"strings"

	"github.com/coder/websocket"
)

type SSHRouter struct {
	mode         string
	dnsTransport *transport.DNSSH
	ctx          context.Context
}

func (r *SSHRouter) handleSSHRelay(conn net.Conn, id string) {
	serverConn, err := GetSSHCon(r.mode, id, r.ctx, r.dnsTransport)
	if err != nil {
		log.Debug().Err(err).Str("Mode", r.mode).Msg("Error relaying SSH connection")

		return
	}
	defer func() {
		_ = serverConn.Close()
	}()
	switch strings.ToLower(r.mode) {
	case "tls", "quic", "dns", "http":
		_, err := serverConn.Write([]byte(id))
		if err != nil {
			log.Debug().Err(err).Msg("Error relaying SSH connection")

			return
		}
	}
	if r.mode == "dns" {
		_, err := serverConn.Write([]byte{'S'})
		if err != nil {
			log.Debug().Err(err).Msg("Error relaying SSH connection")

			return
		}
	}
	relay(conn, serverConn)
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

func (router *SSHRouter) ServeHTTP(w netHttp.ResponseWriter, r *netHttp.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()
	id := r.PathValue("agentId")
	r = commonnet.HTTP10ToHTTP11FakeUpgrader(r)

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
		OriginPatterns:     []string{"*"},
	})
	if err != nil {
		log.Error().Err(err).Str("Mode", "Relay").Str("AgentID", id).Msg("websocket.Accept")

		return
	}
	defer func(wsConn *websocket.Conn) {
		err := wsConn.CloseNow()
		if err != nil {
			log.Warn().Err(err).Str("ID", id).Str("Mode", "Relay").Msg("error closing connection")
		}
	}(wsConn)

	// Convert the websocket connection to a raw net.Conn connection
	conn := websocket.NetConn(ctx, wsConn, websocket.MessageBinary)
	defer func(conn net.Conn) {
		err := conn.Close()
		if err != nil {
			log.Warn().Err(err).Str("ID", id).Err(err).Str("Mode", "Relay").Msg("failed to close websocket connection")
		}
	}(conn)

	router.handleSSHRelay(conn, id)
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
