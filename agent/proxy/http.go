package proxy

import (
	"Goauld/agent/config"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/elazarl/goproxy"
)

type HttpProxy struct {
	Proxy  *goproxy.ProxyHttpServer
	Dialer *ProxyDialer
	Server *http.Server
}

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

	//
	// HTTP Handler
	//
	//	var HttpConnect goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
	//		HTTPConnect := &goproxy.ConnectAction{
	//			Action:    goproxy.ConnectAccept,
	//			TLSConfig: goproxy.TLSConfigFromCA(&goproxy.GoproxyCa),
	//		}
	//
	//		return HTTPConnect, host
	//	}
	//	proxy.OnRequest(goproxy.ReqHostMatches(regexp.MustCompile(".*:80$|.*:8080$"))).HandleConnect(HttpConnect)

	//
	// Connect Handler
	//
	var ConnectHandler goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		fmt.Println("sgohgosngsonsono")
		// HTTPSConnect := &goproxy.ConnectAction{
		// 	// ConnectMitm enables SSL Interception, required for request filtering over HTTPS.
		// 	// Action:    goproxy.ConnectMitm,
		// 	// ConnectAccept preserves upstream SSL Certificates, etc. TCP tunneling basically.
		// 	Action:    goproxy.ConnectAccept,
		// 	TLSConfig: goproxy.TLSConfigFromCA(&goproxy.GoproxyCa),
		// }

		// return HTTPSConnect, host
		return goproxy.OkConnect, host
	}
	proxy.Proxy.OnRequest().HandleConnect(ConnectHandler)
	// proxy.Proxy.OnRequest().HandleConnectFunc(ConnectHandler)

	//
	// Request Handling
	//
	// MITM Action is required for HTTPS Requests (e.g. goproxy.ConnectMitm instead of goproxy.ConnectAccept)
	//
	// proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	// 	log.Fatal(req.URL.String())
	// 	return req, nil
	// })

	srv := &http.Server{
		Handler: proxy.Proxy,
		IdleTimeout: func() time.Duration {
			if timeout, err := time.ParseDuration(os.Getenv("GONTLM_PROXY_IDLE_TIMEOUT")); err == nil {
				return timeout
			} else {
				return 5 * time.Second
			}
		}(),
	}

	proxy.Server = srv
	return proxy
}
