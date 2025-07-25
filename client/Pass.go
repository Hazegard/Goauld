package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"encoding/base64"
	"fmt"
	"os"
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
		serverString := fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost())
		agentString := fmt.Sprintf("%s@%s", agent.Name, agent.Id)

		if strings.Contains(input, agentString) {
			fmt.Println(p.GetStaticPassword(cfg) + agent.SshPasswd)
			return nil
		} else if strings.Contains(input, serverString) {
			fmt.Println(GenerateServerPassword(p.GetStaticPassword(cfg), agent.OneTimePassword))
			return nil
		}
	}

	switch cfg.Pass.Type {
	case "otp":
		fmt.Println(GenerateServerPassword(p.GetStaticPassword(cfg), agent.OneTimePassword))
	case "agent":
		fmt.Println(p.GetStaticPassword(cfg) + agent.SshPasswd)
	default:
		fmt.Printf("OTP:   %s\nAgent: %s\n", GenerateServerPassword(p.GetStaticPassword(cfg), agent.OneTimePassword), agent.SshPasswd)
	}
	return nil
}

func GenerateServerPassword(agentPassword string, serverPassword string) string {
	return fmt.Sprintf("%s|%s", base64.StdEncoding.EncodeToString([]byte(agentPassword)), base64.StdEncoding.EncodeToString([]byte(serverPassword)))
}

func (p *Password) GetStaticPassword(cfg ClientConfig) string {
	if cfg.PrivatePassword != "" {
		return cfg.PrivatePassword
	}
	// If for instance a static password is set but the user currently wants to connect without a password,
	// an empty environment variable "TEALC_PASSWORD=" will be set
	// If we encounter it, we return an empty static password
	empty := prefixEnv("PASSWORD", "")
	for _, e := range os.Environ() {
		if e == empty {
			return ""
		}
	}
	pass, ok := cfg.AgentPassword[cfg.Pass.Agent]
	if ok {
		return pass
	}
	log.Debug().Str("Agent", cfg.Pass.Agent).Msg("No static password found, trying empty static password")
	return ""
}
