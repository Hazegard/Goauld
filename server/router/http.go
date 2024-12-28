package router

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/transport"
	"crypto/tls"
	"errors"
	sio "github.com/karagenc/socket.io-go"
	"net/http"
	"time"
)

type HttpRouter struct {
	controlServer *sio.Server
	wsshHandler   *transport.WSshHandler
	server        *http.Server
	router        *http.ServeMux
	tlsshHandler  *transport.TLSSHServer
	tlsConfig     *tls.Config
}

func NewHttpRouter(
	controlServer *sio.Server,
	wssh *transport.WSshHandler,
	sshttp *transport.SSHttpServer,
	tlssh *transport.TLSSHServer,
) *HttpRouter {

	router := http.NewServeMux()
	router.Handle("/socket.io/", controlServer)
	router.Handle("/wssh/{agentId}", wssh)
	router.Handle("/sshttp/{agentId}", sshttp)
	server := &http.Server{
		//Addr:    config.Get().LocalHttpServer(),
		Handler: router,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise poll requests may fail.
		WriteTimeout: controlServer.HTTPWriteTimeout(),
	}

	httprouter := &HttpRouter{
		controlServer: controlServer,
		wsshHandler:   wssh,
		tlsshHandler:  tlssh,
		server:        server,
		router:        router,
	}

	if config.Get().Tls {
		/*cfg := certmagic.NewDefault()
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
		_, err := cfg.CacheUnmanagedCertificatePEMFile(context.Background(), "./tls_cert.pem", "./key.pem", []string{"*.hazegard.fr", "a.hazegard.fr"})
		if err != nil {
			log.Error().Err(err).Msg("Failed to load TLS certificate")
			return nil
		}

		//cmg := certmagic.New(cache, *cfg)
		tlsconf, err := certmagic.TLS([]string{"*.hazegard.fr", "a.hazegard.fr"})*/

		//tlsconf := cmg.TLSConfig()
		/*		tlsconf.NextProtos = []string{"http/1.1"}
				tlsconf.MinVersion = tls.VersionSSL30*/
		cert, err := tls.LoadX509KeyPair("./tls_cert.pem", "./key.pem")
		if err != nil {
			log.Error().Err(err).Msg("failed to load key pair")
		}
		tlsC := &tls.Config{
			NextProtos:   []string{"http/1.1"},
			Certificates: []tls.Certificate{cert},
		}
		httprouter.tlsConfig = tlsC
	}

	return httprouter

}

func (router *HttpRouter) Serve() error {
	log.Info().Str("Address", config.Get().LocalHttpServer()).Msgf("HTTP server listening")
	var err error

	if config.Get().Tls {
		go router.ServeTLS()
	}
	err = router.server.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
