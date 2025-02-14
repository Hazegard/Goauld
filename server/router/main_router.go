package router

import (
	"crypto/tls"
	"errors"
	"net/http"
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
	router        *http.ServeMux
	tlsshHandler  *transport.TLSSHServer
	tlsConfig     *tls.Config
}

func NewHttpRouter(controlServer *control.SocketIO,
	wssh *transport.WSshHandler,
	sshttp *transport.SSHttpServer,
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
	n.Use(logger)
	n.Use(negroni.NewRecovery())
	n.UseHandler(router)
	server := &http.Server{
		Addr:    config.Get().LocalHttpServer(),
		Handler: n,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise poll requests may fail.
		WriteTimeout: controlServer.Server.HTTPWriteTimeout(),
	}

	httprouter := &MainRouter{
		controlServer: controlServer.Server,
		wsshHandler:   wssh,
		tlsshHandler:  tlssh,
		server:        server,
		router:        router,
	}

	// If the TLS is enabled, configure the server to used TLS
	if config.Get().Tls {
		if config.Get().IsCustomTLS() {
			cert, err := tls.LoadX509KeyPair("./tls_cert.pem", "./key.pem")
			if err != nil {
				log.Error().Err(err).Msg("failed to load key pair")
			}
			tlsC := &tls.Config{
				NextProtos:   []string{"http/1.1"},
				Certificates: []tls.Certificate{cert},
			}
			httprouter.tlsConfig = tlsC
		} else {
			// certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
			tlsConfig, err := certmagic.TLS(config.Get().GetTlsDomains())
			if err != nil {
				return nil, err
			}
			tlsConfig.NextProtos = []string{"http/1.1"}
			tlsConfig.MinVersion = tls.VersionSSL30
			httprouter.tlsConfig = tlsConfig
		}
	}

	return httprouter, nil
}

// Serve serves the Server
func (router *MainRouter) Serve() error {
	log.Info().Str("Address", config.Get().LocalHttpServer()).Msgf("HTTP server listening")
	var err error

	// If the TLS is enabled, run the TLS server in a dedicated goroutine
	if config.Get().Tls {
		go router.ServeTLS()
	}
	// serve the HTTP server
	err = router.server.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
