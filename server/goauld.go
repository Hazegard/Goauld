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
	db, err := persistence.InitDB()
	if err != nil {
		log.Error().Err(err).Msgf("error initializing database")
		return
	}
	agentStore := store.NewAgentStore(db)
	sioServer := control.InitSocketIOServer(agentStore, db)
	wssh := transport.NewWSshHandler(agentStore, db)
	sshttp := transport.NewSSHHttpServer(agentStore, db)
	tlssh := transport.NewTLSSHServer(agentStore, db)
	manageRouter := router.NewUserRouter(db, agentStore)
	r := router.NewHttpRouter(sioServer, wssh, sshttp, tlssh, manageRouter)

	go sshd.StartSshd(ctx, db)
	go func() {
		err := r.Serve()
		if err != nil {
			log.Error().Err(err).Msg("error starting the HTTP server")
		}
	}()

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
