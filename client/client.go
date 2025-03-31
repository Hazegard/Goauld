package main

import (
	"Goauld/client/common"
	"Goauld/client/compiler"
	"fmt"
	"os"

	"Goauld/client/api"
	_common "Goauld/common"
	"Goauld/common/log"
)

func main() {
	if len(os.Args) < 2 {
		// Hijack args if empty to show help if no argument is provided
		//
		os.Args = append(os.Args, "--help")
	}
	if len(os.Args) > 1 && os.Args[1] == "compile" {
		os.Args = os.Args[1:]
		kong, cfg, err := compiler.InitCompilerConfig(APP_NAME, defaultValues)
		if err != nil {
			fmt.Println(err)
			return
		}
		kong.Bind(*cfg)
		err = kong.Run(cfg)
		if err != nil {
			log.Error().Err(err).Msg("error running compiler")
		}
		return
	}
	kong, cfg, err := InitConfig()
	if err != nil {
		fmt.Println(err)
		return
	}
	if cfg.Version {
		fmt.Println(_common.GetVersion())
		return
	}
	if cfg.GenerateConfig {
		cfg.GenerateConfig = false
		c, err := cfg.GenerateYAMLConfig()
		if err != nil {
			log.Error().Err(err).Msg("error generating the agent config")
			return
		}
		fmt.Println(c)
		return
	}
	httpclient := api.NewAPI(cfg.ServerUrl(), cfg.AccessToken, cfg.Insecure)
	kong.Bind(*cfg, httpclient)

	err = kong.Run(httpclient, cfg)
	if err != nil {
		if len(os.Args) > 1 {
			mode := os.Args[1]
			log.Error().Err(err).Str("Mode", mode).Msg("error running " + common.APP_NAME)
			return
		}
		log.Error().Err(err).Msg("error running " + common.APP_NAME)
	}
}
