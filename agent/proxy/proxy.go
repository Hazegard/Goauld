package proxy

import (
	"net/http"
	"net/url"

	"Goauld/agent/config"

	"Goauld/common/log"

	"github.com/aus/proxyplease"
)

// UA the user agent to hide the go user agent.
const UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:136.0) Gecko/20100101 Firefox/136.0"

type userAgentTransport struct {
	rt http.RoundTripper
	ua string
}

// NewHeaderMap returns a map of default HTTP header.
func NewHeaderMap() map[string][]string {
	hm := make(map[string][]string)
	hm["User-Agent"] = []string{UA}

	return hm
}

// RoundTrip perform the HTTP request.
func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// clone so we don’t stomp on callers’ headers
	r2 := req.Clone(req.Context())
	r2.Header.Set("User-Agent", t.ua)

	return t.rt.RoundTrip(r2)
}

// NewProxyDialer return a proxied dialer.
func NewProxyDialer(proxyURL *url.URL, username string, password string, domain string) proxyplease.DialContext {
	proxyplease.SetDebugf(log.ProxyPleaseLog())
	proxy := proxyplease.Proxy{
		TLSConfig: NewTLSConfig(),
	}
	if proxyURL.String() != "" {
		proxy.URL = proxyURL
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

// NewHTTPClientProxy return a new http Client configured to use the proxy.
func NewHTTPClientProxy(tr *http.Transport) *http.Client {
	if tr == nil {
		tr = &http.Transport{}
	}
	tr.TLSClientConfig = NewTLSConfig()
	tr.ForceAttemptHTTP2 = false

	if config.Get().NoProxy() {
		return &http.Client{
			// Transport: &transport,
			Transport: &userAgentTransport{
				rt: tr,
				ua: UA,
			},
		}
	}

	dialContext := NewProxyDialer(config.Get().Proxy(), config.Get().ProxyUsername(), config.Get().ProxyPassword(), config.Get().ProxyDomain())
	tr.DialContext = dialContext

	return &http.Client{
		// Transport: &transport,
		Transport: &userAgentTransport{
			rt: tr,
			ua: UA,
		},
	}
}

// NewTransportProxy returns a new http.Transport configured to use the proxy.
func NewTransportProxy() http.RoundTripper {
	return ProxyTransport(&http.Transport{})
}

// ProxyTransport add the proxy configuration to an existing http.Transport.
//
//nolint:revive
func ProxyTransport(tr *http.Transport) http.RoundTripper {
	if tr == nil {
		tr = &http.Transport{}
	}
	tr.TLSClientConfig = NewTLSConfig()
	tr.ForceAttemptHTTP2 = false

	if config.Get().NoProxy() {
		return &userAgentTransport{
			rt: tr,
			ua: UA,
		}
	}

	dialContext := NewProxyDialer(config.Get().Proxy(), config.Get().ProxyUsername(), config.Get().ProxyPassword(), config.Get().ProxyDomain())
	tr.DialContext = dialContext

	return &userAgentTransport{
		rt: tr,
		ua: UA,
	}
}

// ProxyUsingHttpProxy returns an HTTP transport using the HTTP proxy exposed by the agent.
func ProxyUsingHttpProxy() http.RoundTripper {
	proxyUrl, _ := url.Parse(config.Get().GetLocalHTTPPRoxy())
	internalProxy := NewProxyDialer(proxyUrl, "", "", "")

	tr := &http.Transport{
		DialContext: internalProxy,
	}
	tr.TLSClientConfig = NewTLSConfig()
	tr.ForceAttemptHTTP2 = false

	return tr
}
