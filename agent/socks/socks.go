package socks

import (
	stdlog "log"
	"net"

	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"github.com/things-go/go-socks5"
)

type SocksServer struct {
	socksServer *socks5.Server
	listener    net.Listener
}

// NewSocks returns a new SocksServer
func NewSocks() (*SocksServer, error) {
	defaultLogger := socks5.NewLogger(stdlog.Default())
	options := []socks5.Option{
		socks5.WithLogger(defaultLogger),
	}
	if agent.Get().SocksUseSystemProxy() {
		options = append(options, socks5.WithDial(proxy.NewProxyDialer()))
	}
	s5 := socks5.NewServer(options...)

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
