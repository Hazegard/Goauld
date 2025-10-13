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

// GetTLSConn returns a TLS connection and will abort dialing,
// handshake or header write if ctx is cancelled.
func GetTLSConn(ctx context.Context) (net.Conn, error) {
	// 1) Dial the TCP connection with context
	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", config.Get().TLSURL())
	if err != nil {
		return nil, err
	}

	tlsConf := proxy.NewTLSConfig()
	hostPort := strings.Split(config.Get().TLSURL(), ":")
	tlsConf.ServerName = hostPort[0]
	// 2) Wrap in TLS
	tlsConn := tls.Client(rawConn, tlsConf)

	// 3) Do the TLS handshake with context
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		_ = rawConn.Close()

		return nil, err
	}

	// 4) Write the agent ID header, but bound the write by ctx's deadline
	if dl, ok := ctx.Deadline(); ok {
		// Set a write deadline so if ctx is Done(), Write will fail promptly
		_ = tlsConn.SetWriteDeadline(dl)
	}

	header := []byte(config.Get().ID)
	if _, err := tlsConn.Write(header); err != nil {
		_ = tlsConn.Close()

		return nil, err
	}
	// Clear deadlines so future reads/writes aren’t affected
	_ = tlsConn.SetWriteDeadline(time.Time{})

	return tlsConn, nil
}

/*
// GoodGetTLSConn returns a TLS connection.
func GoodGetTLSConn(ctx context.Context) (net.Conn, error) {
	// Initializes a TLS connection to the server
	conn, err := tls.Dial("tcp", config.Get().TLSURL(), proxy.NewTLSConfig())
	if err != nil {
		return nil, err
	}
	// Write the agent ID as a header to allow the server to identify which agent
	// Write the agent ID as a header to allow the server to identify which agent
	// is currently connecting
	_, err = conn.Write([]byte(config.Get().ID))
	if err != nil {
		return nil, err
	}

	return conn, err
}
*/
