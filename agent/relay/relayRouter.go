package relay

import (
	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
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
	Port          int
	listener      net.Listener
}

func (router *RelayRouter) Init() error {
	addr := fmt.Sprintf(":%d", config.Get().RelayPort())
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	//nolint:forcetypeassert
	port := listener.Addr().(*net.TCPAddr).Port
	router.Port = port
	router.listener = listener
	return nil
}

func (router *RelayRouter) Serve() error {

	return router.Server.Serve(router.listener)
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

	relayer := &RelayRouter{
		ControlRouter: controlRouter,
		SSHRouter:     sshRouter,
		Muxex:         muxer,
		Server:        server,
	}
	err = relayer.Init()
	if err != nil {
		return nil, err
	}
	return relayer, nil
}
