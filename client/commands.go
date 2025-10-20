package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"fmt"
)

type Delete struct {
	Target string `arg:"" name:"agent" help:"The target agent."`
}

func (delete *Delete) Run(clientAPI *api.API, cfg ClientConfig) error {
	err := wrap(clientAPI, cfg, delete.Target, true, true)
	if err != nil {
		log.Error().Err(err).Str("Agent", delete.Target).Msg("Failed to delete agent")
	}
	return err
}

type Reset struct {
	Target string `arg:"" name:"agent" help:"The target agent."`
}

func (reset *Reset) Run(clientAPI *api.API, cfg ClientConfig) error {
	err := wrap(clientAPI, cfg, reset.Target, false, false)
	if err != nil {
		log.Error().Err(err).Str("Agent", reset.Target).Msg("Failed to reset agent")
	}
	return err
}

type Kill struct {
	Target string `arg:"" name:"agent" help:"The target agent."`
}

func (kill *Kill) Run(clientAPI *api.API, cfg ClientConfig) error {
	err := wrap(clientAPI, cfg, kill.Target, true, false)
	if err != nil {
		log.Error().Err(err).Str("Agent", kill.Target).Msg("Failed to kill agent")
	}
	return err
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

func (list *List) Run(clientAPI *api.API, cfg ClientConfig) error {
	agents, err := clientAPI.GetAgents()
	if err != nil {
		log.Error().Err(err).Msg("Failed to list agents")
		return err
	}

	for _, agent := range agents {
		fmt.Println(agent.Name)
	}
	return nil
}
