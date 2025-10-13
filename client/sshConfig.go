package main

import (
	"Goauld/common/log"
	"fmt"
	"os"

	"github.com/kevinburke/ssh_config"
)

func GetFromSSHConfig(sourceFile string, target string) (string, error) {
	//nolint:gosec
	cfgFile, err := os.Open(sourceFile)
	if err != nil {
		log.Error().Str("Agent", target).Err(err).Str("SourceFile", sourceFile).Msg("Failed to open ssh config file")

		return "", fmt.Errorf("failed to open ssh config file: %w", err)
	}
	//nolint:errcheck
	defer cfgFile.Close()
	cfg, err := ssh_config.Decode(cfgFile)
	if err != nil {
		log.Error().Str("Agent", target).Err(err).Str("SourceFile", sourceFile).Msg("Failed to decode ssh config file")

		return "", fmt.Errorf("failed to decode ssh config file: %w", err)
	}

	user, err := cfg.Get(target, "User")
	if err != nil {
		log.Error().Str("Agent", target).Err(err).Str("SourceFile", sourceFile).Msg("Failed to get user from ssh config")

		return "", fmt.Errorf("failed to get user from SSH config file : %w", err)
	}
	name, err := cfg.Get(target, "Hostname")
	if err != nil {
		log.Error().Str("Agent", target).Err(err).Str("SourceFile", sourceFile).Msg("Failed to get hostname from ssh config")

		return "", fmt.Errorf("failed to get hostname from SSH config file : %w", err)
	}

	return fmt.Sprintf("%s@%s", user, name), nil
}
