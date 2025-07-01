package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"fmt"
	"strings"
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

	if len(cfg.Pass.Args) >= 1 {
		input := cfg.Pass.Args[0]
		// fmt.Fprintf(os.Stderr, "%s\n", input)
		serverString := fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost())
		// fmt.Fprintf(os.Stderr, "%s\n", serverString)
		agentString := fmt.Sprintf("%s@%s", agent.Name, agent.Id)
		// fmt.Fprintf(os.Stderr, "%s\n", agentString)
		if strings.Contains(input, agentString) {
			fmt.Println(p.GetStaticPassword(cfg) + agent.SshPasswd)
			// fmt.Fprintf(os.Stderr, "%s\n", p.GetStaticPassword(cfg)+agent.SshPasswd)
			return nil
		} else if strings.Contains(input, serverString) {
			// fmt.Fprintf(os.Stderr, "%s\n", agent.OneTimePassword)
			fmt.Println(agent.OneTimePassword)
			return nil
		}
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
	log.Debug().Str("Agent", cfg.Pass.Agent).Msg("No static password found, trying empty static password")
	return ""
}
