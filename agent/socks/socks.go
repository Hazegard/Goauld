// Package socks holds the socks server
package socks

import (
	"Goauld/agent/proxy"
	"errors"
	"fmt"
	"io"
	stdlog "log"
	"net"
	"net/url"

	"Goauld/common/log"

	"Goauld/agent/config"

	"github.com/things-go/go-socks5"
)

// SocksServer holds the socks server
//
//nolint:revive
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
		if config.Get().HTTPProxyEnabled() {
			u, _ := url.Parse(config.Get().GetLocalHTTPPRoxy())
			options = append(options, socks5.WithDial(proxy.NewProxyDialer(
				u,
				"",
				"",
				"",
			)))
		} else {
			options = append(options, socks5.WithDial(proxy.NewProxyDialer(
				config.Get().SocksProxy(),
				config.Get().SocksProxyUsername(),
				config.Get().SocksProxyPassword(),
				config.Get().SocksProxyDomain(),
			)))
		}
	}
	s5 := socks5.NewServer(options...)

	socksServer := &SocksServer{
		socksServer: s5,
	}

	return socksServer, nil
}

// Serve uses the provided listener to listen and serve the Socks proxy.
func (s *SocksServer) Serve(l net.Listener) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	s.listener = l

	err = s.socksServer.Serve(l)
	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}

// Close closes the socks server.
func (s *SocksServer) Close() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("%v", r)
		}
	}()
	log.Warn().Msgf("Shutting down the socks server")

	return s.listener.Close()
}
