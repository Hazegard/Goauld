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

// NewSocks creates and returns a new SocksServer. It configures the server with a logger and optional proxy dialer.
// If the system proxy is enabled through the configuration, it adds a proxy dialer to the Socks5 server using the
// provided SOCKS proxy settings (proxy address, username, password, and domain). The function initializes a new
// SocksServer instance and returns it along with any potential errors.
func NewSocks() (*SocksServer, error) {
	defaultLogger := socks5.NewLogger(stdlog.Default())
	options := []socks5.Option{
		socks5.WithLogger(defaultLogger),
	}
	if config.Get().SocksUseSystemProxy() {
		options = append(options, socks5.WithDial(proxy.NewProxyDialer(
			config.Get().SocksProxy(),
			config.Get().SocksProxyUsername(),
			config.Get().SocksProxyPassword(),
			config.Get().SocksProxyDomain(),
		)))
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

// Close closes the socks server
func (s *SocksServer) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	log.Warn().Msgf("Shutting done the socks server")
	return s.listener.Close()
}
