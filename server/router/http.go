package router

import (
	"Goauld/server/config"
	"Goauld/server/transport"
	"errors"
	sio "github.com/karagenc/socket.io-go"
	"net/http"
	"time"
)

type HttpRouter struct {
	controlServer *sio.Server
	wsshHandler   transport.WSshHandler
	server        *http.Server
}

func NewHttpRouter(controlServer *sio.Server, handler *transport.WSshHandler, sshttp *transport.SSHttpServer) *HttpRouter {

	router := http.NewServeMux()
	router.Handle("/socket.io/", controlServer)
	router.Handle("/wssh/{agentId}", handler)
	router.Handle("/sshttp/{agentId}", sshttp)
	server := &http.Server{
		Addr:    config.Get().HttpListenAddress,
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
	}

	return httprouter

}

func (router *HttpRouter) Serve() error {
	err := router.server.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
