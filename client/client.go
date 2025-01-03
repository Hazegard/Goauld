package main

import (
	"Goauld/client/api"
	"Goauld/client/config"
	"Goauld/client/tui"
	"Goauld/common/log"
	"fmt"
)

func main() {
	_, cfg, err := config.InitConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	httpclient := api.NewAPI(cfg)
	t := tui.NewTui(httpclient)
	err = t.Run()
	if err != nil {
		log.Error().Err(err).Msg("error running ui")
	}

}
