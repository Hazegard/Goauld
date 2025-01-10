package main

import (
	"fmt"

	"Goauld/client/api"
	"Goauld/common/log"
)

func main() {
	kong, cfg, err := InitConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	httpclient := api.NewAPI(cfg.ServerUrl(), cfg.AccessToken, cfg.Insecure)
	kong.Bind(*cfg, httpclient)

	err = kong.Run(httpclient, cfg)
	if err != nil {
		log.Error().Err(err).Msg("error running ui")
	}
}
