package main

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/control"
	"Goauld/server/persistence"
	"Goauld/server/router"
	"Goauld/server/sshd"
	"Goauld/server/store"
	"Goauld/server/transport"
	"context"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())
	_, _, err := config.InitServer()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the server")
		return
	}
	_db, err := persistence.InitDB()
	if err != nil {
		log.Error().Err(err).Msgf("error initializing database")
		return
	}
	agentStore := store.NewAgentStore(_db)
	sioServer := control.InitSocketIOServer(agentStore, _db)
	wssh := transport.NewWSshHandler(agentStore)
	sshttp := transport.NewSSHHttpServer(agentStore)
	tlssh := transport.NewTLSSHServer(agentStore)
	r := router.NewHttpRouter(sioServer, wssh, sshttp, tlssh)

	go sshd.StartSshd(ctx, _db)
	go r.Serve()

	startSshd(ctx)
	select {
	case <-ctx.Done():
		log.Error().Err(ctx.Err()).Msgf("shutting down")
		cancel()
	}
	<-ctx.Done()
}

func startHttpRouter(r *router.HttpRouter) {

}

func startSshd(ctx context.Context) {

}
