package socks

import (
	"Goauld/agent/proxy"
	"Goauld/common/log"
	"github.com/things-go/go-socks5"
	stdlog "log"
	"net"
)

type SocksServer struct {
	socksServer *socks5.Server
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
func (s *SocksServer) Serve(l net.Listener) {
	go func() {
		defer func(l net.Listener) {
			err := l.Close()
			if err != nil {
				log.Warn().Err(err).Str("Server", "Socks").Msg("close socks server")
			}
		}(l)
		err := s.socksServer.Serve(l)
		if err != nil {
			log.Error().Err(err).Msg("socks server error")
		}
	}()
}
