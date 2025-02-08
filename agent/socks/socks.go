package socks

import (
	"fmt"
	stdlog "log"
	"net"

	"Goauld/common/log"

	"Goauld/agent/config"
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
	if config.Get().SocksUseSystemProxy() {
		options = append(options, socks5.WithDial(proxy.NewProxyDialer(config.Get().Proxy())))
	}
	s5 := socks5.NewServer(options...)

	socksServer := &SocksServer{
		socksServer: s5,
	}
	return socksServer, nil
}

// Serve use the provided listener to listen and serve the Socks proxy
func (s *SocksServer) Serve(l net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	s.listener = l

	return s.socksServer.Serve(l)
}

func (s *SocksServer) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	log.Warn().Msgf("Shutting done the socks server")
	return s.listener.Close()
}
