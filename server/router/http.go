package router

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/transport"
	"crypto/tls"
	"errors"
	sio "github.com/karagenc/socket.io-go"
	"io"
	"net"
	"net/http"
	"time"
)

type HttpRouter struct {
	controlServer *sio.Server
	wsshHandler   transport.WSshHandler
	server        *http.Server
	router        *http.ServeMux
	tlsConfig     *tls.Config
}

func NewHttpRouter(controlServer *sio.Server, handler *transport.WSshHandler, sshttp *transport.SSHttpServer) *HttpRouter {

	router := http.NewServeMux()
	router.Handle("/socket.io/", controlServer)
	router.Handle("/wssh/{agentId}", handler)
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
		wsshHandler:   *handler,
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
	log.Info().Str("Address", config.Get().LocalHttpServer()).Msgf("SSH server listening")
	var err error

	go func() {
		log.Info().Msg("Listening on port 80 for HTTP...")
		err := http.ListenAndServe(":80", router.router)
		if err != nil {
			log.Error().Err(err).Msg("HTTP server failed")
		}
	}()
	if config.Get().Tls {

		//go transport.Test()
		listener, err := tls.Listen("tcp", ":443", router.tlsConfig)
		if err != nil {
			log.Error().Err(err).Msg("Failed to start TLS listener")
			return err
		}
		defer listener.Close()
		log.Info().Msg("Listening on port 443 for HTTPS/RAW TLS...")

		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Warn().Err(err).Msg("Failed to accept connection")
				continue
			}
			go func(c net.Conn) {
				defer c.Close()
				// Check if connection is TLS
				if tlsConn, ok := c.(*tls.Conn); ok {
					if err := tlsConn.Handshake(); err != nil {
						log.Warn().Err(err).Msg("TLS handshake failed")
						return
					}
					state := tlsConn.ConnectionState()
					if state.ServerName == "a.hazegard.fr" {
						// Serve HTTPS traffic
						err := router.server.Serve(NewSingleConnListener(tlsConn))
						if err != nil {
							log.Error().Err(err).Msg("Failed to start server")
						}
					} else if state.ServerName == "b.hazegard.fr" {
						// Serve RAW TLS traffic
						sshConn, err := net.Dial("tcp", config.Get().LocalSShServer())
						if err != nil {
							log.Error().Err(err).Msg("error connecting to server")
							return
						}
						go io.Copy(c, sshConn)
						io.Copy(sshConn, c)
					}
				} else {
					log.Info().Msg("Non-TLS connection received")
				}
			}(conn)
		}
	} else {
		err = router.server.ListenAndServe()
	}
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func handleTlsConn(c net.Conn) {

}

type singleConnListener struct {
	conn      net.Conn
	ch        chan bool
	closeChan chan bool
}

func NewSingleConnListener(conn net.Conn) net.Listener {
	l := &singleConnListener{
		conn:      conn,
		ch:        make(chan bool, 1),
		closeChan: make(chan bool, 1),
	}

	l.ch <- true
	return l
}

// Accept implements net.Listener
func (l *singleConnListener) Accept() (net.Conn, error) {
	select {
	case <-l.ch:
		return l.conn, nil
	case <-l.closeChan:
		return l.conn, http.ErrServerClosed
	}
}

func (l *singleConnListener) Addr() net.Addr {
	return l.conn.LocalAddr()
}

func (l *singleConnListener) Close() error {
	l.closeChan <- true
	return l.conn.Close()
}
