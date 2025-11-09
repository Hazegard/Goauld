package relay

import (
	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"context"
	"fmt"
	"net"
	"net/http"
	"time"
)

type RelayRouter struct {
	ControlRouter *ControlRouter
	SSHRouter     *SSHRouter
	Muxex         *http.ServeMux
	Server        *http.Server
}

func (router *RelayRouter) Serve() error {
	addr := fmt.Sprintf(":%d", config.Get().RelayPort())
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	port := listener.Addr().(*net.TCPAddr).Port
	log.Info().Int("Port", port).Msgf("Relay listening on port %d", port)

	return router.Server.Serve(listener)
}

func NewRelayRouter(ctx context.Context, controlMode string, sshMode string, dnsTransport *transport.DNSSH) (*RelayRouter, error) {
	sio, err := InitSocketIORelayServer(ctx, controlMode, dnsTransport)
	if err != nil {
		return nil, err
	}
	controlRouter := &ControlRouter{
		Server: sio.Server,
	}
	sshRouter := &SSHRouter{
		mode:         sshMode,
		ctx:          ctx,
		dnsTransport: dnsTransport,
	}

	muxer := http.NewServeMux()

	muxer.HandleFunc("/live/{agentId}/{$}", controlRouter.ServeHTTP)
	muxer.Handle("/wssh/{agentId}", sshRouter)

	server := &http.Server{
		Handler: muxer,
		// It is always a good practice to set timeouts.
		ReadTimeout:  120 * time.Second,
		IdleTimeout:  120 * time.Second,
		WriteTimeout: controlRouter.Server.HTTPWriteTimeout(),
	}

	return &RelayRouter{
		ControlRouter: controlRouter,
		SSHRouter:     sshRouter,
		Muxex:         muxer,
		Server:        server,
	}, nil
}
