package transport

import (
	utls "github.com/refraction-networking/utls"
	"net"
)

func NewTlsConn(conn net.Conn) *utls.UConn {
	return utls.UClient(conn, &config, tls.HelloRandomized)
}
