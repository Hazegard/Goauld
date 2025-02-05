package proxy

import (
	"Goauld/agent/config"
	"net/http"
	"net/url"

	"Goauld/common/log"
	"github.com/aus/proxyplease"
)

// NewProxyDialer return a proxified dialer
func NewProxyDialer(proxyUrl *url.URL) proxyplease.DialContext {
	proxyplease.SetDebugf(log.ProxyPleaseLog())
	proxy := proxyplease.Proxy{
		TLSConfig: NewTlsConfig(),
	}
	if proxyUrl.String() != "" {
		proxy.URL = proxyUrl
	}
	return proxyplease.NewDialContext(proxy)
}

// NewHttpClientProxy return a new http Client configured to used the proxy
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

	dialContext := NewProxyDialer(config.Get().Proxy())
	transport.DialContext = dialContext
	return &http.Client{
		Transport: &transport,
	}
}

// NewTransportProxy returns a new http.Transport configured to use the proxy
func NewTransportProxy() *http.Transport {
	return ProxifyTransport(&http.Transport{})
}

// ProxifyTransport add the proxy configuration to an existing http.Transport
func ProxifyTransport(tr *http.Transport) *http.Transport {
	if tr == nil {
		tr = &http.Transport{}
	}
	tr.TLSClientConfig = NewTlsConfig()
	tr.ForceAttemptHTTP2 = false

	if config.Get().NoProxy() {
		return tr
	}

	dialContext := NewProxyDialer(config.Get().Proxy())
	tr.DialContext = dialContext
	return tr
}
