package socks

import (
	"Goauld/agent/proxy"
	"Goauld/common/log"
	"github.com/armon/go-socks5"
	stdlog "log"
	"net"
)

type SocksServer struct {
	socksServer *socks5.Server
}

// NewSocks returns a new SocksServer
func NewSocks() (*SocksServer, error) {
	s5Config := &socks5.Config{
		AuthMethods: nil,
		Credentials: nil,
		Resolver:    nil,
		Rules:       nil,
		Rewriter:    nil,
		BindIP:      nil,
		Logger:      stdlog.Default(),
		Dial:        proxy.NewProxyDialer(),
	}
	s5, err := socks5.New(s5Config)
	if err != nil {
		return nil, err
	}
	socksServer := &SocksServer{
		socksServer: s5,
	}
	return socksServer, nil
}

// Serve use the provided listener to listen and serve the Socks proxy
func (s *SocksServer) Serve(l net.Listener) {
	go func() {
		defer l.Close()
		err := s.socksServer.Serve(l)
		if err != nil {
			log.Error().Err(err).Msg("socks server error")
		}
	}()
}
