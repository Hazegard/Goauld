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
	srvVersion, err := api.Version()
	if err != nil {
		log.Debug().Err(err).Msg("error getting version")
		return
	}
	clientVersion := common.JsonVersion()
	if srvVersion.Compare(clientVersion) != 0 {
		log.Warn().Str("Server", srvVersion.Version).Str("Client", clientVersion.Version).Msg("version mismatch")
		log.Trace().Str("Server Commit", srvVersion.Commit).Str("Client Commit", clientVersion.Commit).Msg("version mismatch")
		log.Trace().Str("Server Date", srvVersion.Date).Str("Client Date", clientVersion.Date).Msg("version mismatch")
	}
}

// dump dumps the information regading the agents currently connected to the server
func (d *Dump) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure, "")
	// CheckAdminVersion(adminApi)

	res, err := adminApi.DumpAll()
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

// Loglevel updates the log level on the server side
func (l *Loglevel) Run(_ *api.API, cfg ClientConfig) error {
	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure, "")
	//CheckAdminVersion(adminApi)
	res, err := adminApi.UpdateLogLevel(cfg.Admin.Loglevel.Level)
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

// Config dumps the server side configuration
func (c *Config) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure, "")
	//CheckAdminVersion(adminApi)
	res, err := adminApi.GetConfig()
	if err != nil {
		return err
	}

	colorYaml.PrintColorizedYAML(res)
	return nil
}

// State dumps the information regading all the agents (connected or not)
func (c *State) Run(_ *api.API, cfg ClientConfig) error {

	adminApi := api.NewAPI(cfg.ServerUrl(), cfg.AdminToken, cfg.Insecure, "")
	//CheckAdminVersion(adminApi)

	res, err := adminApi.DumpState()
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
