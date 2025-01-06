package main

import (
	"Goauld/client/api"
	"Goauld/client/config"
	"Goauld/common/log"
	"fmt"
)

func main() {
	kong, cfg, err := config.InitConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	httpclient := api.NewAPI(cfg.Server, cfg.AccessToken)
	kong.Bind(*cfg, httpclient)

	err = kong.Run(httpclient, cfg)

	if err != nil {
		log.Error().Err(err).Msg("error running ui")
	}

}
