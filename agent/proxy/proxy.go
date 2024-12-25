package proxy

import (
	"github.com/aus/proxyplease"
)

func NewProxyDialer() proxyplease.DialContext {
	return proxyplease.NewDialContext(proxyplease.Proxy{
		URL:              nil,
		Username:         "",
		Password:         "",
		Domain:           "",
		TargetURL:        nil,
		Headers:          nil,
		TLSConfig:        NewTlsConfig(),
		AuthSchemeFilter: nil,
	})
}
