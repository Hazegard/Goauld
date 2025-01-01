package transport

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"net"
)

// GetTlsConn returns a TLS connection
func GetTlsConn(ctx context.Context) (net.Conn, error) {
	// Initializes a TLS connection to the server
	conn, err := tls.Dial("tcp", agent.Get().TlsUrl(), proxy.NewTlsConfig())
	if err != nil {
		return nil, err
	}
	// Write the agent ID as header to allows the server to identify which agent
	// is currently connecting
	_, err = conn.Write([]byte(agent.Get().Id))
	if err != nil {
		return nil, err
	}

	return conn, err
}
