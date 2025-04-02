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

func smuxTLSDialContext(stream *smux.Stream) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		tlsConn := tls.Client(&streamConn{stream}, &tls.Config{
			ServerName:         addr[:len(addr)-5], // strip ":443" from "example.com:443"
			InsecureSkipVerify: true,               // for testing — in prod, validate properly
		})

		// Perform handshake manually
		if err := tlsConn.Handshake(); err != nil {
			return nil, err
		}

		return tlsConn, nil
	}
}

// stream is your *smux.Stream created over the TCP conn
func smuxDialFunc(stream *smux.Stream) func(network, addr string) (net.Conn, error) {
	return func(network, addr string) (net.Conn, error) {
		return &streamConn{stream}, nil
	}
}

func newSmuxHTTPSClient(stream *smux.Stream) *http.Client {
	transport := &http.Transport{
		DialTLSContext: smuxTLSDialContext(stream),
	}

	return &http.Client{
		Transport: transport,
	}
}

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
				InsecureSkipVerify: true, // ❗ for dev/testing only
			})

			if err := tlsConn.Handshake(); err != nil {
				return nil, err
			}

			return tlsConn, nil
		},
	}
}
