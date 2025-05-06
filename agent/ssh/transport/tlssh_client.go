package transport

import (
	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"net"
	"strings"
	"time"
)

// GetTlsConn returns a TLS connection and will abort dialing,
// handshake or header write if ctx is cancelled.
func GetTlsConn(ctx context.Context) (net.Conn, error) {
	// 1) Dial the TCP connection with context
	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", config.Get().TlsUrl())
	if err != nil {
		return nil, err
	}

	tlsConf := proxy.NewTlsConfig()
	hostPort := strings.Split(config.Get().TlsUrl(), ":")
	tlsConf.ServerName = hostPort[0]
	// 2) Wrap in TLS
	tlsConn := tls.Client(rawConn, tlsConf)

	// 3) Do the TLS handshake with context
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		rawConn.Close()
		return nil, err
	}

	// 4) Write the agent ID header, but bound the write by ctx's deadline
	if dl, ok := ctx.Deadline(); ok {
		// Set a write deadline so if ctx is Done(), Write will fail promptly
		_ = tlsConn.SetWriteDeadline(dl)
	}

	header := []byte(config.Get().Id)
	if _, err := tlsConn.Write(header); err != nil {
		tlsConn.Close()
		return nil, err
	}
	// Clear deadlines so future reads/writes aren’t affected
	_ = tlsConn.SetWriteDeadline(time.Time{})

	return tlsConn, nil
}

// GetTlsConn returns a TLS connection
func GoodGetTlsConn(ctx context.Context) (net.Conn, error) {
	// Initializes a TLS connection to the server
	conn, err := tls.Dial("tcp", config.Get().TlsUrl(), proxy.NewTlsConfig())
	if err != nil {
		return nil, err
	}
	// Write the agent ID as a header to allow the server to identify which agent
	// Write the agent ID as a header to allow the server to identify which agent
	// is currently connecting
	_, err = conn.Write([]byte(config.Get().Id))
	if err != nil {
		return nil, err
	}
	return conn, err
}
