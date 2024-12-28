package transport

import (
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"net"
)

func GetTlsConn(ctx context.Context) (net.Conn, error) {
	conn, err := tls.Dial("tcp", "b.hazegard.fr:443", proxy.NewTlsConfig())

	return conn, err
}
