package main

import (
	"Goauld/client/api"
	"Goauld/common/log"
	"fmt"
)

type Clipboard struct {
	Get GetClipboard `cmd:"" name:"get"`
	Set SetClipboard `cmd:"" name:"set"`
}

type GetClipboard struct {
	Target string `arg:""`
}

type SetClipboard struct {
	Target  string `arg:""`
	Content string `arg:""`
}

func (gc *GetClipboard) Run(clientAPI *api.API, cfg ClientConfig) error {
	agent, err := clientAPI.GetAgentByName(cfg.Clipboard.Get.Target)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Target", cfg.VsCode.Target).Msg("Failed to get agent")

		return err
	}

	if agent.HasStaticPassword && cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Error().Err(err).Str("Agent", agent.Name).Msg("Failed to prompt for static password")

			return err
		}
	}
	res, err := clientAPI.GetClipboard(agent.ID, cfg.PrivatePassword)
	if err != nil {
		log.Error().Err(err).Str("Agent", agent.Name).Msg("Failed to get clipboard")

		return err
	}
	//nolint:forbidigo
	fmt.Println(res)

	return nil
}
func (sc *SetClipboard) Run(clientAPI *api.API, cfg ClientConfig) error {
	agent, err := clientAPI.GetAgentByName(cfg.Clipboard.Set.Target)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Target", cfg.VsCode.Target).Msg("Failed to get agent")

		return err
	}
	if agent.HasStaticPassword && cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Error().Err(err).Str("Agent", agent.Name).Msg("Failed to prompt for static password")

			return err
		}
	}
	err = clientAPI.SetClipboard(agent.ID, cfg.PrivatePassword, cfg.Clipboard.Set.Content)

	return err
}
