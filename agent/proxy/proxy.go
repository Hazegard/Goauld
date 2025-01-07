package proxy

import (
	"net/http"

	"Goauld/common/log"
	"github.com/aus/proxyplease"
)

// NewProxyDialer return a proxified dialer
func NewProxyDialer() proxyplease.DialContext {
	proxyplease.SetDebugf(log.ProxyPleaseLog())
	return proxyplease.NewDialContext(proxyplease.Proxy{
		TLSConfig: NewTlsConfig(),
	})
}

// NewHttpClientProxy return a new http Client configured to used the proxy
func NewHttpClientProxy() *http.Client {
	dialContext := NewProxyDialer()
	httpclient := &http.Client{
		Transport: &http.Transport{
			DialContext:       dialContext,
			TLSClientConfig:   NewTlsConfig(),
			ForceAttemptHTTP2: false,
		},
	}
	return httpclient
}

// NewTransportProxy returns a new http.Transport configured to use the proxy
func NewTransportProxy() *http.Transport {
	return ProxifyTransport(&http.Transport{})
}

// ProxifyTransport add the proxy configuration to an existing http.Transport
func ProxifyTransport(tr *http.Transport) *http.Transport {
	dialContext := NewProxyDialer()
	if tr == nil {
		tr = &http.Transport{}
	}
	tr.DialContext = dialContext
	tr.TLSClientConfig = NewTlsConfig()
	tr.ForceAttemptHTTP2 = false

	return tr
}
