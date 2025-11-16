// Package main holds the server entrypoint
package main

import (
	"Goauld/common"
	"context"
	"fmt"

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
	// Initialize the server configuration from the command line, environment variable and configuration files
	_, _, err := config.InitServer()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the server")

		return
	}
	if config.Get().Version {
		//nolint:forbidigo
		fmt.Println(common.GetVersion())

		return
	}

	if config.Get().GenerateConfig {
		c, err := config.Get().GenerateYAMLConfig()
		if err != nil {
			log.Error().Err(err).Msg("error generating the server config")

			return
		}
		//nolint:forbidigo
		fmt.Println(c)

		return
	}
	// Initialize the database
	db, err := persistence.InitDB()
	if err != nil {
		log.Error().Err(err).Msgf("error initializing database")

		return
	}
	// Reset the connected status of all agents when restarting
	err = db.ResetAgents()
	if err != nil {
		log.Error().Err(err).Msgf("error reseting agents")
	}
	// Initialize all the required components
	agentStore := store.NewAgentStore(db)
	sioServer, err := control.InitSocketIOServer(agentStore, db)
	if err != nil {
		log.Error().Err(err).Msgf("error initializing socketio server")

		return
	}
	wssh := transport.NewWSshHandler(agentStore, db)
	sshttp, err := transport.NewSSHHttpServer(agentStore, db)
	if err != nil {
		log.Error().Err(err).Msgf("error initializing ssh http server")
	}
	tlssh := transport.NewTLSSHServer(agentStore, db)
	manageRouter := router.NewManageRouter(db, agentStore)
	adminRouter := router.NewAdminRouter(db, agentStore)
	staticRouter := router.NewStaticRouter()
	quicRouter := transport.NewQUICServer(agentStore, db)
	sshttp2 := transport.NewServer(
		"",
		"",
		"",
		false,
		true,
		false,
		config.Get().HTTPDomain[0],
		config.Get().LocalSSHAddr(),
		"",
		"",
		db,
	)

	// Initialize the HTTP router
	r, err := router.NewHTTPRouter(sioServer, wssh, sshttp, tlssh, manageRouter, adminRouter, staticRouter, quicRouter, sshttp2)
	if err != nil {
		log.Error().Err(err).Msgf("error initializing http router")

		return
	}

	// Initialize and start the SSHD server
	go sshd.StartSshd(ctx, db, agentStore)
	go func() {
		// Start the HTTP server
		err := r.Serve()
		if err != nil {
			log.Error().Err(err).Msg("error starting the HTTP server")
		}
	}()

	if config.Get().DNS {
		log.Info().Msg("starting DNS server")
		go func() {
			dnsServer, err := transport.NewDNSSHServer(agentStore, db)
			if err != nil {
				log.Error().Err(err).Msgf("error initializing dns server")

				return
			}
			err = dnsServer.Run()
			if err != nil {
				log.Error().Err(err).Msg("error starting the DNS server")
			}
		}()
	}

	// waits for the end
	<-ctx.Done()
	log.Error().Err(ctx.Err()).Msgf("shutting down")
	cancel()
}
