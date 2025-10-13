// Package main holds the client entrypoint
package main

import (
	"Goauld/client/api"
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

// dump dumps the information regading the agents currently connected to the server.
func (d *Dump) Run(_ *api.API, cfg ClientConfig) error {
	adminAPA := api.NewAPI(cfg.ServerURL(), cfg.AdminToken, cfg.Insecure, "")

	res, err := adminAPA.DumpAll()
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

// Loglevel updates the log level on the server side.
func (l *Loglevel) Run(_ *api.API, cfg ClientConfig) error {
	adminAPA := api.NewAPI(cfg.ServerURL(), cfg.AdminToken, cfg.Insecure, "")
	res, err := adminAPA.UpdateLogLevel(cfg.Admin.Loglevel.Level)
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

// Config dumps the server side configuration.
func (c *Config) Run(_ *api.API, cfg ClientConfig) error {
	adminAPA := api.NewAPI(cfg.ServerURL(), cfg.AdminToken, cfg.Insecure, "")
	res, err := adminAPA.GetConfig()
	if err != nil {
		return err
	}

	colorYaml.PrintColorizedYAML(res)

	return nil
}

// State dumps the information regading all the agents (connected or not).
func (c *State) Run(_ *api.API, cfg ClientConfig) error {
	adminAPA := api.NewAPI(cfg.ServerURL(), cfg.AdminToken, cfg.Insecure, "")

	res, err := adminAPA.DumpState()
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
