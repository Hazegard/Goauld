// Package main holds the server entrypoint
package main

import (
	"Goauld/common"
	"Goauld/server/server"
	"context"
	"fmt"

	"Goauld/common/log"
	"Goauld/server/config"
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
	err = server.Run(ctx, cancel)
	if err != nil {
		log.Error().Err(err).Msg("error starting the server")
	}
}
