package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"fmt"
)

type Password struct {
	Agent string   `name:"agent" yaml:"agent" help:"Agent name to retrieve password."`
	Type  string   `name:"type" yaml:"type" help:"Password to retrieve (OTP/Agent)."`
	Args  []string `arg:"" optional:""`
}

// Run executes the pass subcommand
func (p *Password) Run(api *api.API, cfg ClientConfig) error {

	if len(cfg.Pass.Args) > 0 && cfg.Pass.Agent == "" {
		cfg.Pass.Agent = cfg.Pass.Args[0]
	}
	agent, err := api.GetAgentByName(cfg.Pass.Agent)
	if err != nil {
		return err
	}
	switch cfg.Pass.Type {
	case "otp":
		fmt.Println(agent.OneTimePassword)
	case "agent":
		fmt.Println(p.GetStaticPassword(cfg) + agent.SshPasswd)
	default:
		fmt.Printf("OTP:   %s\nAgent: %s\n", agent.OneTimePassword, agent.SshPasswd)
	}
	return nil
}

func (p *Password) GetStaticPassword(cfg ClientConfig) string {
	if cfg.PrivatePassword != "" {
		return cfg.PrivatePassword
	}
	pass, ok := cfg.AgentPassword[cfg.Pass.Agent]
	if ok {
		return pass
	}
	log.Warn().Str("Agent", cfg.Pass.Agent).Msg("No static password found, trying empty static password")
	return ""
}
