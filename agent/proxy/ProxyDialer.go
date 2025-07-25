package proxy

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	"bufio"
	"context"
	"errors"
	"github.com/aus/proxyplease"
	"github.com/jellydator/ttlcache/v2"
	"golang.org/x/sync/singleflight"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

var ProxyDialerCacheTimeout = 60 * time.Minute

type ProxyDialer struct {
	group          *singleflight.Group
	cache          *ttlcache.Cache
	directDialer   func(ctx context.Context, network string, address string) (net.Conn, error)
	ProxyOverrides map[string]*url.URL
}

// NewHttpProxyDialer returns a new ProxyDialer instance
func NewHttpProxyDialer() *ProxyDialer {
	//
	// LRU Cache: Memoize DialContexts for 60 minutes
	//
	dialerCache := ttlcache.NewCache()
	_ = dialerCache.SetTTL(ProxyDialerCacheTimeout)
	dialerCacheGroup := singleflight.Group{}

	return &ProxyDialer{
		group:          &dialerCacheGroup,
		cache:          dialerCache,
		directDialer:   new(net.Dialer).DialContext,
		ProxyOverrides: make(map[string]*url.URL),
	}
}

// ProxyDialer returns a proxyplease.DialContext that manages proxy connections, either using a direct connection
// or a proxy specified by the ProxyOverrides map. The function checks if the address has a cached dial context,
// and if not, it constructs one based on the provided scheme and address. It supports exact and suffix matching
// for proxy overrides and also handles tunneling for secure connections. The context is cached for future use
// to improve performance by reusing previously established connections. The method also handles the connection
// through proxy authentication (username, password, domain) and sends the CONNECT request for tunneling when necessary.
func (p *ProxyDialer) ProxyDialer(scheme, addr string, pxyUrl *url.URL) proxyplease.DialContext {
	log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")
	cacheKey := addr
	if pxyUrl != nil && pxyUrl.Host != "" && p.ProxyOverrides == nil {
		cacheKey = pxyUrl.Host
	}

	log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")

	if dctx, err := p.cache.Get(cacheKey); err == nil {
		return dctx.(proxyplease.DialContext)
	}
	log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")

	dctx, err, _ := p.group.Do(cacheKey, func() (pxyCtx interface{}, err error) {
		if p.ProxyOverrides != nil {
			var detected bool
			hosts := []string{addr, strings.Split(addr, ":")[0]}
			//
			// Exact Match
			//
			for _, host := range hosts {
				if pxy, ok := p.ProxyOverrides[strings.ToLower(host)]; ok {
					// If empty (nil) assume direct connection
					if pxy == nil {
						return p.directDialer, nil
					}

					detected = true
					pxyUrl = pxy
					break
				}
			}

			//
			// Suffix Match
			//
			if !detected {
				for _, host := range hosts {
					for dns, pxy := range p.ProxyOverrides {
						if strings.HasSuffix(strings.ToLower(host), dns) {
							// If empty (nil) assume direct connection
							if pxy == nil {
								return p.directDialer, nil
							}
							detected = true
							pxyUrl = pxy
							break
						}
					}
					if detected {
						break
					}
				}
			}

			//
			// Check if we need to tunnel
			//
			if detected {
				if tunnelPxy, ok := p.ProxyOverrides[pxyUrl.Host]; ok {
					var tunnelctx proxyplease.DialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
						conn, err := proxyplease.NewDialContext(proxyplease.Proxy{
							URL:      tunnelPxy,
							Username: config.Get().HttpProxyUsername(),
							Password: config.Get().HttpProxyPassword(),
							Domain:   config.Get().HttpProxyDomain(),
						})(ctx, network, pxyUrl.Host)

						if err != nil {
							return conn, err
						}

						req := &http.Request{
							Method: http.MethodConnect,
							URL:    &url.URL{Opaque: addr, Scheme: scheme},
							Host:   addr,
							Header: http.Header{
								"Proxy-Connection": []string{"Keep-Alive"},
							},
						}

						// req.Write(os.Stdout)
						br := bufio.NewReader(conn)
						if err = req.Write(conn); err != nil {
							return conn, err
						}

						resp, err := http.ReadResponse(br, req)
						_ = resp.Body.Close()
						if err != nil {
							return conn, err
						}

						if resp.StatusCode == http.StatusOK {
							return conn, nil
						}
						return conn, errors.New(resp.Status)
					}
					_ = p.cache.Set(cacheKey, tunnelctx)
					return tunnelctx, nil
				}
			}
		}

		log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")
		pxyCtx = proxyplease.NewDialContext(proxyplease.Proxy{
			URL:       pxyUrl,
			Username:  config.Get().HttpProxyUsername(),
			Password:  config.Get().HttpProxyPassword(),
			Domain:    config.Get().HttpProxyDomain(),
			TargetURL: &url.URL{Host: addr, Scheme: scheme},
		})

		err = p.cache.Set(cacheKey, pxyCtx)

		log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")
		return
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to proxy")
	}
	log.Debug().Str("scheme", scheme).Str("addr", addr).Msg("proxy dialer")

	return dctx.(proxyplease.DialContext)
}
