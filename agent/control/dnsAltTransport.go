package control

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"strings"

	sio "github.com/hazegard/socket.io-go"
	eio "github.com/hazegard/socket.io-go/engine.io"
	"github.com/coder/websocket"
)

// InitOverDNS tries to connect to the control plan using the DNS transport.
func (cpc *ControlPlanClient) InitOverDNSAlt(conn net.Conn, success chan<- struct{}, chanErr chan<- error) error {
	// DNS MODE means we are using http to simplify the exchanges
	u := strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoURL(config.Get().ID), "https://"), "http://")
	cpc.url = "http://" + u

	cfg := getDNSAltEioConfig(conn)

	return cpc.init(cfg, success, chanErr)
}

// getEioConfig return the socket.io underlying configuration.
func getDNSAltEioConfig(conn net.Conn) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Str("Transport", transportName).Msg("Client transport upgrade done")
			},
			HTTPTransport: NewDnsAltTransport(conn),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: newDnsAltHTTPandHTTPSClient(conn),
			},
			// When tunneling over DNS, if we use polling only or polling then websocket upgrade,
			// The tunnel fails to establish properly as the server responds to unwanted content to the open HTTP socket.
			// Here we use the full duplex websocket mechanism to ensure that the tunnel is properly working
			// On the client side
			Transports: []string{"websocket"},
		},
	}
}

// NewSmuxTransport creates and returns an *http.Transport that uses a provided smux stream
// for both non-TLS (HTTP) and TLS (HTTPS) connections. The function defines custom dialers for each protocol:
// one for HTTP that directly uses the smux stream, and one for HTTPS that establishes a TLS connection
// with the stream.
func NewDnsAltTransport(conn net.Conn) *http.Transport {
	return &http.Transport{
		DialContext: func(_ context.Context, _, _ string) (net.Conn, error) {
			return conn, nil
		},

		DialTLSContext: func(_ context.Context, _, addr string) (net.Conn, error) {
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				// fallback for addr without port
				host = strings.Split(addr, ":")[0]
			}

			tlsConn := tls.Client(conn, &tls.Config{
				ServerName: host,
				//nolint:gosec
				InsecureSkipVerify: true,
			})

			if err := tlsConn.Handshake(); err != nil {
				return nil, err
			}

			return tlsConn, nil
		},
	}
}

// newSmuxHTTPandHTTPSClient creates and returns an HTTP client that supports both HTTP and HTTPS protocols
// over an existing smux stream. It uses custom dial functions for HTTP and TLS connections over the stream.
// The function returns a configured *http.Client with a custom Transport that handles non-TLS and TLS connections.
// This client is used in DNS only mode as the HTTP traffic must go within a DNS tunnel.
func newDnsAltHTTPandHTTPSClient(conn net.Conn) *http.Client {
	dialContext := func(_ context.Context, _, _ string) (net.Conn, error) {
		return conn, nil
	}

	dialTLSContext := func(_ context.Context, _, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(conn, &tls.Config{
			ServerName: host,
			//nolint:gosec
			InsecureSkipVerify: true, // for testing only!
		})

		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}

	transport := &http.Transport{
		DialContext:    dialContext,
		DialTLSContext: dialTLSContext,
	}

	return &http.Client{Transport: transport}
}
