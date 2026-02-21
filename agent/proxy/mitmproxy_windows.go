//go:build windows

package proxy

import (
	"Goauld/common/log"
	"crypto/tls"
	"crypto/x509"
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
	proxyLogger := logger{l: log.Get().With().Str("From", "MITM HttpProxy").Logger()}
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

	mitm := goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(LoadCert())}

	var AlwaysMitm goproxy.FuncHttpsHandler = func(host string, ctx *goproxy.ProxyCtx) (*goproxy.ConnectAction, string) {
		return &mitm, host
	}
	proxy.Proxy.OnRequest().HandleConnect(AlwaysMitm)
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

func LoadCert() *tls.Certificate {
	var GoproxyCa tls.Certificate
	// When we included the embedded certificate inside this file, we made
	// sure that it was valid.
	// If there is an error here, this is a really exceptional case that requires
	// a panic. It should NEVER happen!
	var err error
	GoproxyCa, err = tls.X509KeyPair(CA_CERT, CA_KEY)
	if err != nil {
		panic("Error parsing builtin CA: " + err.Error())
	}

	if GoproxyCa.Leaf, err = x509.ParseCertificate(GoproxyCa.Certificate[0]); err != nil {
		panic("Error parsing builtin CA leaf: " + err.Error())
	}
	return &GoproxyCa
}

var CA_CERT = []byte(`-----BEGIN CERTIFICATE-----
MIIFAzCCAuugAwIBAgIUcAeRqw9CibzhOzewXOZOHfKQGoIwDQYJKoZIhvcNAQEN
BQAwETEPMA0GA1UEAwwGR29hdWxkMB4XDTI2MDIyMTIwMzE0M1oXDTM2MDIxOTIw
MzE0M1owETEPMA0GA1UEAwwGR29hdWxkMIICIjANBgkqhkiG9w0BAQEFAAOCAg8A
MIICCgKCAgEAzReBxb+dAEzUIt3dHHLo+1HgI/MG9Z4d34gGCSzH4lqFAcLcT3ZQ
JAjIQwKsoFnPFrH3yI9m5awySTBiJ9b/eR32E/KAtWlcz+KbZIIGlSv+LC5quHoU
XASCKcwpI/BkJk8Mz/c+FuxLfSlrXgphW5P5cm2FAye5coAvRTogncr+PgTu+x2i
Rh2SSVPvs0tERSxTW4WQAE8ne9Ih18H9BhkP1Ib6GyauxJwdDLnXgGLTOdtRp46u
mZG5WBoFn8pFD4OcJI63rvd5pUSa2YOtxxjJRcqvUdv5tB1eI7Mc0urfM8irqLVh
1WIpdhHiCPjAI3JMLs0UDw3wnK3wiMPMX9qDSQkff6sEe1BikRgH2W2TZgX+vbgl
nSJqdAWW//1wlsojLiiMQa0udbU8HeZmL8mJOmsb800eg3ybR6h3cs8i0MD/Q+j7
3uhrA7/zhoIuk6PjrviXJ4wxHmyWK/a8RcxXqVRIp0KUTL/AAQDJvntfPOYwGL5j
/m+P4o0NcVIKycBTsj/8IIwOn69fFVBuN1bX4Qpx0inLgBG8FwA/QBJB3WtpU6q6
bQqDXY9kGSSzYgpJlnCKjer/8QSju6/+1x3mRVjMDkoYOJ02bGcy7OaAFep45MID
2crrwWdo2yHBjMHvsXQvG8fBDtpUxlbqpab1wpkHYeuhZO9ADLPCoCMCAwEAAaNT
MFEwHQYDVR0OBBYEFJCSpF8l7FlmJ+jZrvKD0fHbAO03MB8GA1UdIwQYMBaAFJCS
pF8l7FlmJ+jZrvKD0fHbAO03MA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQEN
BQADggIBAHtux7vyUcoImcf1bGB/Qu4Ly5sY9/dMZ+ypS4UnqJ7+r84n8yO0irrX
V5MaFvrklEsBeoHCCvumIkaH/BD/+VKlF2Ih/pKuXnFJ9qCzSGjxV4IbnD/9wBQx
26WYVOblIO1+AojBKoAq7txo8tdzI2FN5j1IewwL7/d2koathjwTOsvz7QyL13KP
Lfn/dDsM/hPZ6Ca58zAjmdwj8n6ibILG68ZxwV8zUcQ6tHYx2sPqi9lSlxi5PRX7
4Fj21i2Zovj99SdDsoVQByuWxLDuGm3hYzkvbJDmnQ4/v1hO7NGoH05HvBNevpDm
FHMNZngC7VJL+9OVP3Sq81bmXViFLvKajWfCWZkkZKCuus+xQ//SArNSfeybgkaA
g+g38xqgz9/EDdzf6lCzISUq7HXxLnIZAFCZT4Nh6a8YKL4HIgmn9lUt1sxNSViQ
cztccJgycwWDkxP6nU3/aC9rpr+m9wN0le6ZmesxGG2L9a6eGvgl4wDYuYEIC9zS
KUX6Q9wHdVk7UX5uuktABLpsMlrjsJRHB+l8n1d+Dru1N8qPOHa83RbjOlx2yZH/
EJfiA2+q2em1tRYobKriOaWYj1SnBqu1n3FV3mXGaywEUuFxWHvS8JPo1emJyM1S
FunR4anr/hrKQT2KBedJLSQ5VV4AwXi60wnVMNgsMh4ENRT4wDSE
-----END CERTIFICATE-----`)

var CA_KEY = []byte(`-----BEGIN PRIVATE KEY-----
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
bS8d1nDT4QSn4tt5CczET3RUgaL679c=
-----END PRIVATE KEY-----`)
