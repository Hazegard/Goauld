package proxy

import (
	"Goauld/agent/config"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/elazarl/goproxy"
)

type HttpProxy struct {
	Proxy  *goproxy.ProxyHttpServer
	Dialer *ProxyDialer
	Server *http.Server
}

// InitHttpProxy initializes and returns a configured HttpProxy instance. The function sets up a proxy server
// using the goproxy library with custom dialers for HTTP and HTTPS connections. It configures various proxy
// settings such as connection limits, verbose logging, and headers to be kept across requests. The function
// also handles CONNECT requests for HTTPS connections, allowing the proxy to route the traffic properly.
func InitHttpProxy() *HttpProxy {

	proxy := &HttpProxy{
		Proxy:  goproxy.NewProxyHttpServer(),
		Dialer: NewHttpProxyDialer(),
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

	// HTTP
	proxy.Proxy.Tr.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
		return proxy.Dialer.ProxyDialer("http", addr, config.Get().HttpProxy())(ctx, network, addr)
	}

	// HTTPS
	proxy.Proxy.ConnectDialWithReq = func(req *http.Request, network, addr string) (net.Conn, error) {
		fmt.Printf(">> CONNECT dial to %s (%s)\n", addr, network)
		conn, err := proxy.Dialer.ProxyDialer("https", addr, config.Get().HttpProxy())(req.Context(), network, addr)
		fmt.Printf("<< CONNECT dial done (err: %v)\n", err)
		return conn, err
	}

	var ConnectHandler goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {

		return goproxy.OkConnect, host
	}
	proxy.Proxy.OnRequest().HandleConnect(ConnectHandler)

	srv := &http.Server{
		Handler: proxy.Proxy,
		IdleTimeout: func() time.Duration {
			return 5 * time.Second
		}(),
	}

	proxy.Server = srv
	return proxy
}
