package main

import (
	"Goauld/client/api"
	"Goauld/common"
	"Goauld/common/log"
	colorYaml "Goauld/common/yaml"
	"github.com/goccy/go-yaml"
)

type Admin struct {
	Dump     Dump     `cmd:""`
	Loglevel Loglevel `cmd:""`
	Config   Config   `cmd:""`
	State    State    `cmd:""`
}

type Dump struct{}
type Config struct{}

type State struct {
}
type Loglevel struct {
	Level string `arg:"" help:"Log level"`
}

func CheckAdminVersion(api *api.API) {
	err, srvVersion := api.AdminVersion()
	if err != nil {
		log.Warn().Err(err).Msg("error getting version")
		return
	}
	clientVersion := common.JsonVersion()
	if srvVersion.Compare(clientVersion) != 0 {
		log.Warn().Str("Server", srvVersion.Version).Str("Client", clientVersion.Version).Msg("version mismatch")
		log.Trace().Str("Server Commit", srvVersion.Commit).Str("Client Commit", clientVersion.Commit).Msg("version mismatch")
		log.Trace().Str("Server Date", srvVersion.Date).Str("Client Date", clientVersion.Date).Msg("version mismatch")
	}
}

func (d *Dump) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)
	CheckAdminVersion(adminApi)

	err, res := adminApi.DumpAll()
	if err != nil {
		return err
	}

	y, err := yaml.Marshal(res)
	if err != nil {
		return err
	}
	colorYaml.PrintColorizedYAML(string(y))
	return nil
}

func (l *Loglevel) Run(_ *api.API, cfg ClientConfig) error {
	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)
	CheckAdminVersion(adminApi)
	err, res := adminApi.UpdateLogLevel(cfg.Admin.Loglevel.Level)
	if err != nil {
		return err
	}
	y, err := yaml.Marshal(res)
	if err != nil {
		return err
	}
	colorYaml.PrintColorizedYAML(string(y))
	return nil
}

func (c *Config) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)
	CheckAdminVersion(adminApi)
	err, res := adminApi.GetConfig()
	if err != nil {
		return err
	}

	colorYaml.PrintColorizedYAML(res)
	return nil
}

func (c *State) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure)
	CheckAdminVersion(adminApi)

	err, res := adminApi.DumpState()
	if err != nil {
		return err
	}

	state, err := yaml.Marshal(res)
	if err != nil {
		return err
	}

	colorYaml.PrintColorizedYAML(string(state))
	return nil
}
