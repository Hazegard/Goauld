//go:build !mini
// +build !mini

package proxy

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	"context"
	"net"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
)

// HTTPProxy holds the HTTP proxy.
type HTTPProxy struct {
	Proxy  *goproxy.ProxyHttpServer
	Dialer *ProxyDialer
	Server *http.Server
}

// InitHTTPProxy initializes and returns a configured HTTPProxy instance. The function sets up a proxy server
// using the goproxy library with custom dialers for HTTP and HTTPS connections. It configures various proxy
// settings such as connection limits, verbose logging, and headers to be kept across requests. The function
// also handles CONNECT requests for HTTPS connections, allowing the proxy to route the traffic properly.
func InitHTTPProxy() *HTTPProxy {
	proxy := &HTTPProxy{
		Proxy:  goproxy.NewProxyHttpServer(),
		Dialer: NewHTTPProxyDialer(),
	}
	//
	// Proxy DialContexts
	//
	proxy.Proxy.Tr.Proxy = nil
	proxy.Proxy.Tr.MaxIdleConnsPerHost = 10
	proxy.Proxy.Verbose = true
	proxy.Proxy.AllowHTTP2 = false
	proxy.Proxy.KeepAcceptEncoding = true
	proxy.Proxy.KeepHeader = true
	proxy.Proxy.KeepDestinationHeaders = true
	proxyLogger := log.Get().With().Str("From", "HttpProxy").Logger()
	proxy.Proxy.Logger = &proxyLogger

	// HTTP
	proxy.Proxy.Tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return proxy.Dialer.ProxyDialer("http", addr, config.Get().HTTPProxy())(ctx, network, addr)
	}

	// HTTPS
	proxy.Proxy.ConnectDialWithReq = func(req *http.Request, network, addr string) (net.Conn, error) {
		log.Trace().Str("Addr", addr).Str("Network", network).Msg("CONNECT DIAL")
		conn, err := proxy.Dialer.ProxyDialer("https", addr, config.Get().HTTPProxy())(req.Context(), network, addr)
		if err != nil {
			log.Debug().Err(err).Str("Addr", addr).Str("Network", network).Msg("CONNECT DIAL ERROR DONE")
		}

		return conn, err
	}

	proxy.Proxy.OnResponse().DoFunc(func(resp *http.Response, _ *goproxy.ProxyCtx) *http.Response {
		return resp
	})

	var ConnectHandler goproxy.FuncHttpsHandler = func(host string, _ *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return goproxy.OkConnect, host
	}
	proxy.Proxy.OnRequest().HandleConnect(ConnectHandler)

	srv := &http.Server{
		Handler: proxy.Proxy,
		IdleTimeout: func() time.Duration {
			return 5 * time.Second
		}(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	proxy.Server = srv

	return proxy
}
