package transport

import (
	"context"
	"crypto/tls"
	"net"

	"Goauld/agent/config"
	"Goauld/agent/proxy"
)

// GetTlsConn returns a TLS connection
func GetTlsConn(ctx context.Context) (net.Conn, error) {
	// Initializes a TLS connection to the server
	conn, err := tls.Dial("tcp", config.Get().TlsUrl(), proxy.NewTlsConfig())
	if err != nil {
		return nil, err
	}
	// Write the agent ID as header to allow the server to identify which agent
	// is currently connecting
	_, err = conn.Write([]byte(config.Get().Id))
	if err != nil {
		return nil, err
	}

	return conn, err
}
