package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"Goauld/common/types"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
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

	staticPwd, err := p.GetStaticPassword(cfg)
	if err != nil {
		return err
	}
	if len(cfg.Pass.Args) >= 1 {
		input := cfg.Pass.Args[0]
		serverString := fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost())
		agentString := fmt.Sprintf("%s@%s", agent.Name, agent.Id)

		if strings.Contains(input, agentString) {
			fmt.Println(staticPwd + agent.SshPasswd)
			return nil
		} else if strings.Contains(input, serverString) {
			fmt.Println(GenerateServerPassword(staticPwd, agent.OneTimePassword))
			return nil
		}
	}

	switch cfg.Pass.Type {
	case "otp":
		fmt.Println(GenerateServerPassword(staticPwd, agent.OneTimePassword))
	case "agent":
		fmt.Println(staticPwd + agent.SshPasswd)
	default:
		fmt.Printf("OTP:   %s\nAgent: %s\n", GenerateServerPassword(staticPwd, agent.OneTimePassword), agent.SshPasswd)
	}
	return nil
}

func GenerateServerPassword(agentPassword string, serverPassword string) string {
	hash, _ := bcrypt.GenerateFromPassword([]byte(agentPassword), bcrypt.DefaultCost)
	res, err := json.Marshal(types.NewServerToAGentPassword(string(hash), serverPassword))
	if err != nil {
		return ""
	}
	return string(res)
}

func (p *Password) GetStaticPassword(cfg ClientConfig) (string, error) {
	pwd := p.getStaticPassword(cfg)
	if len(pwd) > 72 {
		return "", bcrypt.ErrPasswordTooLong
	}
	return pwd, nil
}
func (p *Password) getStaticPassword(cfg ClientConfig) string {
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
