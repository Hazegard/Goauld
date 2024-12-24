package main

import (
	"Goauld/agent/agent"
	"Goauld/agent/control"
	"Goauld/agent/sshd"
	"Goauld/common/log"
	"context"
)

func main() {
	controlErr := make(chan error)
	sshErr := make(chan error)

	_, err := agent.InitAgent()
	if err != nil {
		log.Error().Err(err).Msg("error initializing the agent")
		return
	}

	ctx := context.Background()
	go func() {
		controlErr <- control.NewClient(ctx)
	}()

	go func() {
		sshErr <- sshd.StartSShd()
	}()

	select {
	case err := <-controlErr:
		log.Error().Err(err).Msg("error starting the agent")
	case err := <-sshErr:
		log.Error().Err(err).Msg("error starting the sshd server")
	}

}
