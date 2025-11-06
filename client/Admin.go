// Package main holds the client entrypoint
package main

import (
	"Goauld/client/api"
	colorYaml "Goauld/common/yaml"

	"github.com/goccy/go-yaml"
)

type Admin struct {
	Dump     Dump     `cmd:"" name:"dump" yaml:"dump" help:"Dump all the agent information."`
	Loglevel Loglevel `cmd:"" name:"log-level" yaml:"log-level" help:"Update the server log level."`
	Config   Config   `cmd:"" name:"config" yaml:"config" help:"Display the server configuration"`
	State    State    `cmd:"" name:"state" yaml:"state" help:"Display the full server state (agents, configuration,etc...)"`
}

type Dump struct{}
type Config struct{}

type State struct {
}
type Loglevel struct {
	Level string `arg:"" name:"level" yaml:"level" help:"Log level"`
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
