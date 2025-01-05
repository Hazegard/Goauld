package socks

import (
	"Goauld/agent/proxy"
	"github.com/things-go/go-socks5"
	stdlog "log"
	"net"
)

type SocksServer struct {
	socksServer *socks5.Server
	listener    net.Listener
}

// NewSocks returns a new SocksServer
func NewSocks() (*SocksServer, error) {

	defaultLogger := socks5.NewLogger(stdlog.Default())
	s5 := socks5.NewServer(socks5.WithLogger(defaultLogger), socks5.WithDial(proxy.NewProxyDialer()))

	socksServer := &SocksServer{
		socksServer: s5,
	}
	return socksServer, nil
}

// Serve use the provided listener to listen and serve the Socks proxy
func (s *SocksServer) Serve(l net.Listener) error {
	s.listener = l

	return s.socksServer.Serve(l)
}

func (s *SocksServer) Close() error {
	return s.listener.Close()
}
