package router

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/quic-go/quic-go/http3"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/control"
	"Goauld/server/transport"

	"github.com/caddyserver/certmagic"

	sio "github.com/hazegard/socket.io-go"
	"github.com/urfave/negroni"
)

// MainRouter is the primary router that listens to.
type MainRouter struct {
	controlServer *sio.Server
	wsshHandler   *transport.WSshHandler
	server        *http.Server
	server3       *http3.Server
	router        *http.ServeMux
	tlsshHandler  *transport.TLSSHServer
	tlsConfig     *tls.Config
	quicSSH       *transport.QUICServer
	sshttp2       *transport.Server
}

// NewHTTPRouter returns an HTTP router responsible for handling all HTTP related connections.
func NewHTTPRouter(controlServer *control.SocketIO,
	wssh *transport.WSshHandler,
	sshttp *transport.SSHHttpServer,
	tlssh *transport.TLSSHServer,
	manageRouter *ManageRouter,
	adminRouter *AdminRouter,
	staticRouter *StaticRouter,
	quicSSH *transport.QUICServer,
	sshttp2 *transport.Server,
) (*MainRouter, error) {
	// Initializing the router and adding the handlers to paths
	router := http.NewServeMux()
	router.HandleFunc("/live/{agentId}/{$}", controlServer.ServeHTTP)
	router.Handle("/wssh/{agentId}", wssh)
	router.Handle("/sshttp/{agentId}", sshttp)
	router.Handle("/manage/", http.StripPrefix("/manage", manageRouter.GetRouter()))
	router.Handle("/admin/", http.StripPrefix("/admin", adminRouter.GetRouter()))
	router.Handle("/binaries/", http.StripPrefix("/binaries", staticRouter.GetRouter()))

	// Negroni allow using middleware, such as logger and recovery mechanism
	n := negroni.New()
	logger := negroni.NewLogger()
	logger.SetFormat("{{.StartTime}} | {{.Status}} | \t {{.Duration}} | {{.Hostname}} | {{.Method}} {{.Path}} | {{.Request.RemoteAddr}}")
	logger.ALogger = log.GetNegroniLogger()
	// Custom middleware
	n.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if (strings.HasPrefix(r.URL.Path, "/sshttp") && r.Method == http.MethodPost) ||
			(strings.HasPrefix(r.URL.Path, "/sshttp2") && r.Method == http.MethodGet) ||
			(strings.HasPrefix(r.URL.Path, "/wssh/00000000000000000000000000000000") && r.Method == http.MethodGet) ||
			(strings.HasPrefix(r.URL.Path, "/live/00000000000000000000000000000000") && r.Method == http.MethodGet) {
			next(w, r)

			return
		}

		logger.ServeHTTP(w, r, next)
	})
	// n.Use(logger)
	n.Use(negroni.NewRecovery())
	n.UseHandler(router)
	server := &http.Server{
		// Addr:    config.Get().LocalHTTPAddr(),
		Handler: n,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise, poll requests may fail.
		WriteTimeout: controlServer.Server.HTTPWriteTimeout(),
	}

	server3 := &http3.Server{
		Handler: n,
	}

	router.HandleFunc("/sshttp2/{agentId}/", sshttp2.HandleRequest)

	httprouter := &MainRouter{
		controlServer: controlServer.Server,
		wsshHandler:   wssh,
		tlsshHandler:  tlssh,
		server:        server,
		server3:       server3,
		router:        router,
		quicSSH:       quicSSH,
		sshttp2:       sshttp2,
	}

	// If the TLS is enabled, configure the server to use TLS
	if config.Get().TLS {
		if config.Get().IsCustomTLS() {
			certLoader := certReloader{}
			err := certLoader.Load(config.Get().TLSCert, config.Get().TLSKey)
			// cert, err := tls.LoadX509KeyPair(config.Get().TLSCert, config.Get().TLSKey)

			if err != nil {
				log.Error().Err(err).Msg("failed to load key pair")
			}
			//nolint:gosec
			tlsC := &tls.Config{
				NextProtos: []string{"http/1.1"},
				//Certificates: []tls.Certificate{cert},
				//nolint:staticcheck,gosec // SA1019
				MinVersion:     tls.VersionSSL30,
				GetCertificate: certLoader.GetCertificate,
			}
			httprouter.tlsConfig = tlsC
			quicTLS := &tls.Config{
				NextProtos: []string{"quic"},
				//Certificates: []tls.Certificate{cert},
				MinVersion:     tls.VersionTLS13,
				GetCertificate: certLoader.GetCertificate,
			}

			// Reload on SIGHUP (common in Unix deployments)
			go func() {
				ch := make(chan os.Signal, 1)
				signal.Notify(ch, syscall.SIGHUP)
				for range ch {
					if err := certLoader.Load(config.Get().TLSCert, config.Get().TLSKey); err != nil {
						log.Printf("TLS reload failed: %v", err)

						continue
					}
					log.Printf("TLS certificate reloaded")
				}
			}()

			httprouter.server3.TLSConfig = quicTLS
		} else {
			// certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
			certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
			certmagic.DefaultACME.Agreed = true
			certmagic.DefaultACME.Email = config.Get().LetsEncryptMail
			certmagic.DefaultACME.DisableHTTPChallenge = false

			var allowedDomains atomic.Value

			renew := func(domains []string) {
				m := make(map[string]struct{}, len(domains))
				for _, domain := range domains {
					m[domain] = struct{}{}
				}
				allowedDomains.Store(m)
			}
			renew(config.Get().GetTLSDomains())

			certMagicConfig := certmagic.NewDefault()
			certMagicConfig.OnDemand = &certmagic.OnDemandConfig{
				DecisionFunc: func(_ context.Context, name string) error {
					v := allowedDomains.Load()
					if v == nil {
						return fmt.Errorf("no certificate loaded for domain %s", name)
					}
					m, ok := v.(map[string]struct{})
					if !ok {
						return fmt.Errorf("unexpected type %T for domain %s", v, name)
					}
					_, ok = m[name]
					if ok {
						return nil
					}

					return fmt.Errorf("domain %s not allowed", name)
				},
			}

			baseConfig, err := certmagic.TLS(config.Get().GetTLSDomains())
			if err != nil {
				return nil, err
			}
			//nolint:staticcheck,gosec // SA1019
			baseConfig.MinVersion = tls.VersionSSL30

			baseConfig.GetCertificate = certMagicConfig.GetCertificate

			tlsConfig := baseConfig.Clone()
			tlsConfig.NextProtos = append([]string{"http/1.1"}, tlsConfig.NextProtos...)
			httprouter.tlsConfig = tlsConfig

			quicConfig := baseConfig.Clone()
			quicConfig.NextProtos = append([]string{"quic", "ssh", "h3"}, quicConfig.NextProtos...)
			httprouter.server3.TLSConfig = quicConfig

			// Reload on SIGHUP (common in Unix deployments)
			go func() {
				ch := make(chan os.Signal, 1)
				signal.Notify(ch, syscall.SIGHUP)
				for range ch {
					err := config.Get().ReloadDomains()
					if err != nil {
						log.Error().Err(err).Msg("failed to reload domains")

						continue
					}
					renew(config.Get().GetTLSDomains())
					log.Printf("TLS certificate reloaded")
				}
			}()
		}
	}

	return httprouter, nil
}

// Serve serves the Server.
func (router *MainRouter) Serve() error {
	var err error

	// If the TLS is enabled, run the TLS server in a dedicated goroutine
	if config.Get().TLS {
		go router.ServeTLS()

		if config.Get().Quic {
			go router.ServeQUIC()
		}
	}
	// serve the HTTP server
	listener, err := net.Listen("tcp", config.Get().LocalHTTPAddr())
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	config.Get().UpdateHTTPAddr(listener.Addr().(*net.TCPAddr).Port)

	log.Info().Str("Address", config.Get().LocalHTTPAddr()).Msgf("HTTP server listening")
	err = router.server.Serve(listener)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}

	return nil
}

type certReloader struct {
	cert atomic.Value
}

func (r *certReloader) Load(certFile string, keyFile string) error {
	c, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return err
	}
	r.cert.Store(c)

	return nil
}

func (r *certReloader) GetCertificate(*tls.ClientHelloInfo) (*tls.Certificate, error) {
	v := r.cert.Load()
	if v == nil {
		return nil, errors.New("no certificate loaded")
	}
	c, ok := v.(*tls.Certificate)
	if !ok {
		return nil, errors.New("invalid certificate")
	}

	return c, nil
}
