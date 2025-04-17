package router

import (
	"crypto/tls"
	"errors"
	"github.com/quic-go/quic-go/http3"
	"net"
	"net/http"
	"strings"
	"time"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/control"
	"Goauld/server/transport"
	"github.com/caddyserver/certmagic"

	sio "github.com/karagenc/socket.io-go"
	"github.com/urfave/negroni"
)

// MainRouter is the primary router that listen on
type MainRouter struct {
	controlServer *sio.Server
	wsshHandler   *transport.WSshHandler
	server        *http.Server
	server3       *http3.Server
	router        *http.ServeMux
	tlsshHandler  *transport.TLSSHServer
	tlsConfig     *tls.Config
	quickSSH      *transport.QUICKServer
}

func NewHttpRouter(controlServer *control.SocketIO,
	wssh *transport.WSshHandler,
	sshttp *transport.SSHHttpServer,
	tlssh *transport.TLSSHServer,
	manageRouter *ManageRouter,
	adminRouter *AdminRouter,
	staticRouter *StaticRouter,
) (*MainRouter, error) {
	// Initializing the router and adding the handlers to paths
	router := http.NewServeMux()
	router.Handle("/socket.io/", controlServer.Server)
	router.Handle("/wssh/{agentId}", wssh)
	router.Handle("/sshttp/{agentId}", sshttp)
	router.Handle("/manage/", http.StripPrefix("/manage", manageRouter.GetRouter()))
	router.Handle("/admin/", http.StripPrefix("/admin", adminRouter.GetRouter()))
	router.Handle("/binaries/", http.StripPrefix("/binaries", staticRouter.GetRouter()))

	// Negroni allow to used middleware, such as logger and recovery mecanism
	n := negroni.New()
	logger := negroni.NewLogger()
	logger.ALogger = log.GetNegroniLogger()
	// Custom middleware
	n.UseFunc(func(w http.ResponseWriter, r *http.Request, next http.HandlerFunc) {
		if strings.HasPrefix(r.URL.Path, "/sshttp") {
			next(w, r)
			return
		}

		logger.ServeHTTP(w, r, next)
	})
	// n.Use(logger)
	n.Use(negroni.NewRecovery())
	n.UseHandler(router)
	server := &http.Server{
		// Addr:    config.Get().LocalHttpAddr(),
		Handler: n,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise poll requests may fail.
		WriteTimeout: controlServer.Server.HTTPWriteTimeout(),
	}

	server3 := &http3.Server{
		Handler: n,
	}

	httprouter := &MainRouter{
		controlServer: controlServer.Server,
		wsshHandler:   wssh,
		tlsshHandler:  tlssh,
		server:        server,
		server3:       server3,
		router:        router,
	}

	// If the TLS is enabled, configure the server to used TLS
	if config.Get().Tls {
		if config.Get().IsCustomTLS() {
			cert, err := tls.LoadX509KeyPair(config.Get().TlsCert, config.Get().TlsKey)
			if err != nil {
				log.Error().Err(err).Msg("failed to load key pair")
			}
			tlsC := &tls.Config{
				NextProtos:   []string{"http/1.1"},
				Certificates: []tls.Certificate{cert},
				MinVersion:   tls.VersionSSL30,
			}
			httprouter.tlsConfig = tlsC
			httprouter.server3.TLSConfig = tlsC
		} else {
			// certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
			certmagic.DefaultACME.CA = certmagic.LetsEncryptProductionCA
			certmagic.DefaultACME.Agreed = true
			certmagic.DefaultACME.Email = "mail@example.com"

			tlsConfig, err := certmagic.TLS(config.Get().GetTlsDomains())
			if err != nil {
				return nil, err
			}
			tlsConfig.NextProtos = []string{"http/1.1"}
			tlsConfig.MinVersion = tls.VersionSSL30
			httprouter.tlsConfig = tlsConfig
			httprouter.server3.TLSConfig = tlsConfig
		}

	}

	return httprouter, nil
}

// Serve serves the Server
func (router *MainRouter) Serve() error {
	var err error

	// If the TLS is enabled, run the TLS server in a dedicated goroutine
	if config.Get().Tls {
		go router.ServeTLS()
	}
	// serve the HTTP server
	listener, err := net.Listen("tcp", config.Get().LocalHttpAddr())
	if err != nil {
		return err
	}
	config.Get().UpdateHTTPAddr(listener.Addr().(*net.TCPAddr).Port)

	log.Info().Str("Address", config.Get().LocalHttpAddr()).Msgf("HTTP server listening")
	err = router.server.Serve(listener)

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
