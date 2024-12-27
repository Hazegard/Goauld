package proxy

import (
	"github.com/aus/proxyplease"
	"net/http"
)

func NewProxyDialer() proxyplease.DialContext {
	return proxyplease.NewDialContext(proxyplease.Proxy{
		TLSConfig: NewTlsConfig(),
	})
}

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

func NewTransportProxy() *http.Transport {
	return ProxifyTransport(&http.Transport{})
}

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
