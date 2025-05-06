package control

import (
	"context"
	"crypto/tls"
	"github.com/xtaci/smux"
	"net"
	"net/http"
	"strings"
	"time"
)

// The DNSTransport is used to tunnel the control socket when running in DNS-only mode.
// Indeed, when we have to tunnel the SSH traffic using DNS, it means that the control socket that relies
// On socket.io cannot reach the control server over HTTP/Websocket.

type streamConn struct {
	*smux.Stream
}

func (s *streamConn) LocalAddr() net.Addr                { return dummyAddr("smux-local") }
func (s *streamConn) RemoteAddr() net.Addr               { return dummyAddr("smux-remote") }
func (s *streamConn) SetDeadline(t time.Time) error      { return s.Stream.SetDeadline(t) }
func (s *streamConn) SetReadDeadline(t time.Time) error  { return s.Stream.SetReadDeadline(t) }
func (s *streamConn) SetWriteDeadline(t time.Time) error { return s.Stream.SetWriteDeadline(t) }

type dummyAddr string

func (d dummyAddr) Network() string { return string(d) }
func (d dummyAddr) String() string  { return string(d) }

// newSmuxHTTPandHTTPSClient creates and returns an HTTP client that supports both HTTP and HTTPS protocols
// over an existing smux stream. It uses custom dial functions for HTTP and TLS connections over the stream.
// The function returns a configured *http.Client with a custom Transport that handles non-TLS and TLS connections.
// This client is used in DNS only mode as the HTTP traffic must go within a DNS tunnel
func newSmuxHTTPandHTTPSClient(stream *smux.Stream) *http.Client {
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return &streamConn{stream}, nil
	}

	dialTLSContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}

		tlsConn := tls.Client(&streamConn{stream}, &tls.Config{
			ServerName:         host,
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

// NewSmuxTransport creates and returns an *http.Transport that uses a provided smux stream
// for both non-TLS (HTTP) and TLS (HTTPS) connections. The function defines custom dialers for each protocol:
// one for HTTP that directly uses the smux stream, and one for HTTPS that establishes a TLS connection
// with the stream.
func NewSmuxTransport(stream *smux.Stream) *http.Transport {
	return &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return &streamConn{stream}, nil
		},

		DialTLSContext: func(ctx context.Context, network, addr string) (net.Conn, error) {

			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				// fallback for addr without port
				host = strings.Split(addr, ":")[0]
			}

			tlsConn := tls.Client(&streamConn{stream}, &tls.Config{
				ServerName:         host,
				InsecureSkipVerify: true,
			})

			if err := tlsConn.Handshake(); err != nil {
				return nil, err
			}

			return tlsConn, nil
		},
	}
}
