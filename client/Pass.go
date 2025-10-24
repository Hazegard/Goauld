package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"Goauld/common/types"
	"encoding/json"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

type Password struct {
	Agent string   `name:"agent" yaml:"agent" help:"Agent name to retrieve password."`
	Type  string   `name:"type" yaml:"type" help:"Password to retrieve (OTP/Agent)."`
	Args  []string `arg:"" optional:""`
}

// Run executes the pass subcommand.
func (p *Password) Run(clientAPI *api.API, cfg ClientConfig) error {
	if len(cfg.Pass.Args) > 0 && cfg.Pass.Agent == "" {
		cfg.Pass.Agent = cfg.Pass.Args[0]
	}
	log.Trace().Str("agent", cfg.Pass.Agent).Str("type", cfg.Pass.Type).Msg("getting password")
	agent, err := clientAPI.GetAgentByName(cfg.Pass.Agent)
	if err != nil {
		log.Error().Err(err).Str("agent", cfg.Pass.Agent).Msg("failed to get agent")

		return err
	}

	staticPwd, err := p.GetStaticPassword(cfg)
	if err != nil {
		return err
	}
	if len(cfg.Pass.Args) >= 1 {
		input := cfg.Pass.Args[0]
		serverString := fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost())
		agentString := fmt.Sprintf("%s@%s", agent.Name, agent.ID)

		if strings.Contains(input, agentString) {
			//nolint:forbidigo
			fmt.Println(staticPwd + agent.SSHPasswd)

			return nil
		} else if strings.Contains(input, serverString) {
			//nolint:forbidigo
			fmt.Println(GenerateServerPassword(staticPwd, agent.OneTimePassword))

			return nil
		}
	}

	switch cfg.Pass.Type {
	case "otp":
		//nolint:forbidigo
		fmt.Println(GenerateServerPassword(staticPwd, agent.OneTimePassword))
	case "agent":
		//nolint:forbidigo
		fmt.Println(staticPwd + agent.SSHPasswd)
	default:
		//nolint:forbidigo
		fmt.Printf("OTP:   %s\nAgent: %s\n", GenerateServerPassword(staticPwd, agent.OneTimePassword), agent.SSHPasswd)
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
	pwd := cfg.GetStaticPassword()
	if len(pwd) > 72 {
		return "", bcrypt.ErrPasswordTooLong
	}

	return pwd, nil
}
