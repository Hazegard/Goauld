package proxy

import (
	"net/http"
	"net/url"

	"Goauld/agent/config"

	"Goauld/common/log"
	"github.com/aus/proxyplease"
)

// NewProxyDialer return a proxied dialer
func NewProxyDialer(proxyUrl *url.URL, username string, password string, domain string) proxyplease.DialContext {
	proxyplease.SetDebugf(log.ProxyPleaseLog())
	proxy := proxyplease.Proxy{
		TLSConfig: NewTlsConfig(),
	}
	if proxyUrl.String() != "" {
		proxy.URL = proxyUrl
	}
	if username != "" {
		proxy.Username = username
	}
	if password != "" {
		proxy.Password = password
	}
	if domain != "" {
		proxy.Domain = domain
	}
	return proxyplease.NewDialContext(proxy)
}

// NewHttpClientProxy return a new http Client configured to use the proxy
func NewHttpClientProxy() *http.Client {
	transport := http.Transport{
		TLSClientConfig:   NewTlsConfig(),
		ForceAttemptHTTP2: false,
	}

	if config.Get().NoProxy() {
		return &http.Client{
			Transport: &transport,
		}
	}

	dialContext := NewProxyDialer(config.Get().Proxy(), config.Get().ProxyUsername(), config.Get().ProxyPassword(), config.Get().ProxyDomain())
	transport.DialContext = dialContext
	return &http.Client{
		Transport: &transport,
	}
}

// NewTransportProxy returns a new http.Transport configured to use the proxy
func NewTransportProxy() *http.Transport {
	return ProxyTransport(&http.Transport{})
}

// ProxyTransport add the proxy configuration to an existing http.Transport
func ProxyTransport(tr *http.Transport) *http.Transport {
	if tr == nil {
		tr = &http.Transport{}
	}
	tr.TLSClientConfig = NewTlsConfig()
	tr.ForceAttemptHTTP2 = false

	if config.Get().NoProxy() {
		return tr
	}

	dialContext := NewProxyDialer(config.Get().Proxy(), config.Get().ProxyUsername(), config.Get().ProxyPassword(), config.Get().ProxyDomain())
	tr.DialContext = dialContext
	return tr
}
