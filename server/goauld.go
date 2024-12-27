package main

import (
	"Goauld/common/log"
	"Goauld/server/control"
	"Goauld/server/db"
	"Goauld/server/router"
	"Goauld/server/sshd"
	"Goauld/server/store"
	"Goauld/server/transport"
	"context"
)

func main() {

	ctx, cancel := context.WithCancel(context.Background())

	db, err := db.InitDB()
	if err != nil {
		log.Error().Err(err).Msgf("error initializing database")
		return
	}
	agentStore := store.NewAgentStore()
	sioServer := control.InitSocketIOServer(agentStore)
	wssh := transport.NewWSshHandler(agentStore, db)
	sshttp := transport.InitSSHTTPServer(agentStore)
	r := router.NewHttpRouter(sioServer, wssh, sshttp)

	go sshd.StartSshd(ctx, db)
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
