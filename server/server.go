package main

import (
	"context"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/control"
	"Goauld/server/persistence"
	"Goauld/server/router"
	"Goauld/server/sshd"
	"Goauld/server/store"
	"Goauld/server/transport"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	// Initialize the server configuration from command line, environment variable and configuration files
	_, _, err := config.InitServer()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the server")
		return
	}
	// Initialize the database
	db, err := persistence.InitDB()
	if err != nil {
		log.Error().Err(err).Msgf("error initializing database")
		return
	}
	// Initialize all the required components
	agentStore := store.NewAgentStore(db)
	sioServer, err := control.InitSocketIOServer(agentStore, db)
	if err != nil {
		log.Error().Err(err).Msgf("error initializing socketio server")
		return
	}
	wssh := transport.NewWSshHandler(agentStore, db)
	sshttp := transport.NewSSHHttpServer(agentStore, db)
	tlssh := transport.NewTLSSHServer(agentStore, db)
	manageRouter := router.NewManageRouter(db, agentStore)
	adminRouter := router.NewAdminRouter(db, agentStore)

	// Initialize the HTTP router
	r := router.NewHttpRouter(sioServer, wssh, sshttp, tlssh, manageRouter, adminRouter)

	// Initialize and start the SSHD server
	go sshd.StartSshd(ctx, db)
	go func() {
		// Start the HTTP server
		err := r.Serve()
		if err != nil {
			log.Error().Err(err).Msg("error starting the HTTP server")
		}
	}()

	// waits for the end
	select {
	case <-ctx.Done():
		log.Error().Err(ctx.Err()).Msgf("shutting down")
		cancel()
	}
	<-ctx.Done()
}
