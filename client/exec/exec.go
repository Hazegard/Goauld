package exec

import (
	"Goauld/client/api"
	"Goauld/client/config"
	"fmt"
)

func Run(api api.API, cfg config.ClientConfig, agentName string) (string, error) {
	agent, err := api.GetAgentByName(agentName)
	if err != nil {
		return "", err
	}
	fmt.Printf("%+v\n", agent)
	return agent.SshPasswd, nil
}
