//go:build windows

package proxy

import (
	"Goauld/common/log"
	"net/http"
	"sync"
	"time"

	"github.com/elazarl/goproxy"
)

// MITMHTTPProxy holds the HTTP proxy that performs mitm to inject NTLM/Kerberos authentication.
type MITMHTTPProxy struct {
	Proxy    *goproxy.ProxyHttpServer
	Dialer   *ProxyDialer
	Server   *http.Server
	Username string
	Password string //nolint:gosec
	Domain   string
}

// InitMITMHTTPProxy initializes and returns a configured MITMHTTPProxy instance.
// It intercepts all communications to inject if required NTLM / Kerberos authentication using the underlying credentials
func InitMITMHTTPProxy(u string, p string, d string) (*MITMHTTPProxy, error) {
	proxy := &MITMHTTPProxy{
		Proxy:    goproxy.NewProxyHttpServer(),
		Dialer:   NewHTTPProxyDialer(),
		Domain:   d,
		Password: p,
		Username: u,
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
	proxyLogger := log.Get().With().Str("From", "MITM HttpProxy").Logger()
	proxy.Proxy.Logger = &proxyLogger

	sspiTransport := &SSPITransport{
		Base:                ProxyUsingHttpProxy(),
		Domain:              proxy.Domain,
		Username:            proxy.Username,
		Password:            proxy.Password,
		RespectExistingAuth: true,
		mu:                  sync.Mutex{},
	}

	proxy.Proxy.OnResponse().DoFunc(func(resp *http.Response, _ *goproxy.ProxyCtx) *http.Response {
		return resp
	})

	proxy.Proxy.OnRequest().HandleConnect(goproxy.AlwaysMitm)
	proxy.Proxy.OnRequest().DoFunc(func(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
		res, err := sspiTransport.RoundTrip(req)
		if err != nil {
			log.Debug().Err(err).Msg("MITM CONNECT ERROR")
		}
		return req, res
	})

	srv := &http.Server{
		Handler: proxy.Proxy,
		IdleTimeout: func() time.Duration {
			return 5 * time.Second
		}(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	proxy.Server = srv

	return proxy, nil
}
