package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"
	"fmt"
)

type Delete struct {
	Target string `arg:"" name:"agent" yaml:"agent" help:"Target agent to delete."`
}

func (d *Delete) Run(clientAPI *api.API, cfg ClientConfig) error {
	err := wrap(clientAPI, cfg, d.Target, true, true)
	if err != nil {
		log.Error().Err(err).Str("Agent", d.Target).Msg("Failed to d agent")
	}

	return err
}

type Reset struct {
	Target string `arg:"" name:"agent" yaml:"agent" help:"Target agent to reset."`
}

func (reset *Reset) Run(clientAPI *api.API, cfg ClientConfig) error {
	clientSSH, err := NewCustomSSH(clientAPI, cfg, cfg.Reset.Target)
	if err != nil {
		return err
	}
	status, _, err := clientSSH.SSHClient.SendRequest(_ssh.Restart, false, nil)
	if err != nil {
		return err
	}
	if status {
		log.Info().Str("Agent", cfg.Reset.Target).Msg("Agent restarted")

		return nil
	}
	log.Error().Msgf("Failed to reset agent: %s", cfg.Reset.Target)

	return nil

	/*err := wrap(clientAPI, cfg, reset.Target, false, false)
	if err != nil {
		log.Error().Err(err).Str("Agent", reset.Target).Msg("Failed to reset agent")
	}

	return err*/
}

type Kill struct {
	Target string `arg:"" name:"agent" yaml:"agent"  help:"Target agent to terminate."`
	Delete bool   `name:"delete" yaml:"delete" help:"Also delete the agent’s binary after termination."`
}

func (kill *Kill) Run(clientAPI *api.API, cfg ClientConfig) error {
	clientSSH, err := NewCustomSSH(clientAPI, cfg, cfg.Reset.Target)
	if err != nil {
		return err
	}
	req := _ssh.Kill
	if kill.Delete {
		req = _ssh.Delete
	}
	status, _, err := clientSSH.SSHClient.SendRequest(req, false, nil)
	if err != nil {
		return err
	}
	if status {
		log.Info().Str("Agent", cfg.Reset.Target).Msg("Agent killed")

		return nil
	}
	log.Error().Msgf("Failed to kill agent: %s", cfg.Reset.Target)

	return nil
	/*err := wrap(clientAPI, cfg, kill.Target, true, false)
	if err != nil {
		log.Error().Err(err).Str("Agent", kill.Target).Msg("Failed to kill agent")
	}

	return err*/
}

func wrap(clientAPI *api.API, cfg ClientConfig, target string, doExit bool, doDelete bool) error {
	agent, err := clientAPI.GetAgentByName(target)
	if err != nil {
		log.Error().Err(err).Msgf("Failed to get agent")
	}

	if agent.HasStaticPassword && cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Error().Err(err).Str("Agent", agent.Name).Msg("Failed to prompt for static password")

			return err
		}
	}

	return clientAPI.KillAgent(agent.ID, doExit, doDelete, cfg.PrivatePassword)
}

type List struct {
}

func (list *List) Run(clientAPI *api.API, _ ClientConfig) error {
	agents, err := clientAPI.GetAgents()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list agents")

		return err
	}

	for _, agent := range agents {
		//nolint:forbidigo
		fmt.Println(agent.Name)
	}

	return nil
}
