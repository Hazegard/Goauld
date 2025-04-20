package transport

import (
	"context"
	"crypto/tls"
	"net"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/proxy"
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

	// 2) Wrap in TLS
	tlsConn := tls.Client(rawConn, proxy.NewTlsConfig())

	// 3) Do the TLS handshake with context
	if err := tlsConn.HandshakeContext(ctx); err != nil {
		rawConn.Close()
		return nil, err
	}

	// 4) Write the agent ID header, but bound the write by ctx's deadline
	if dl, ok := ctx.Deadline(); ok {
		// Set a write deadline so if ctx is Done(), Write will fail promptly
		tlsConn.SetWriteDeadline(dl)
	}

	header := []byte(config.Get().Id)
	if _, err := tlsConn.Write(header); err != nil {
		tlsConn.Close()
		return nil, err
	}
	// Clear deadlines so future reads/writes aren’t affected
	tlsConn.SetWriteDeadline(time.Time{})

	return tlsConn, nil
}
