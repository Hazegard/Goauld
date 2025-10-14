package main

import (
	"Goauld/client/api"
	"Goauld/client/types"
	"Goauld/common/log"
	"Goauld/common/utils"
	"Goauld/common/vscode"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

type VsCode struct {
	Target     string `arg:""`
	RemotePath string `arg:"" default:"."`
}

func (v VsCode) Run(clientAPI *api.API, cfg ClientConfig) error {
	err := CheckVsCode()
	if err != nil {
		log.Error().Err(err).Msg("VSCode is not installed")

		return err
	}

	configDir := GetConfigDir()
	userVsCode := filepath.Join(configDir, "User")
	log.Debug().Str("Agent", cfg.VsCode.Target).Str("ConfigDir", userVsCode).Str("Path", userVsCode).Msg("Generated User VsCode config")
	err = os.MkdirAll(userVsCode, 0o750)
	if err != nil {
		log.Error().Err(err).Str("Path", userVsCode).Msg("Failed to create User VsCode directory")

		return err
	}

	sshConfigFile := filepath.Join(userVsCode, "ssh_config")
	sshConfig := GenSSHConfig(cfg.VsCode.Target)
	log.Debug().Str("Agent", cfg.VsCode.Target).Str("Config", sshConfig).Str("Path", sshConfigFile).Msg("Generated SSH config file")
	err = utils.WriteToFile(sshConfig, sshConfigFile)
	if err != nil {
		log.Error().Str("Agent", cfg.VsCode.Target).Err(err).Str("Path", sshConfigFile).Msg("Failed to write SSH config file")

		return fmt.Errorf("failed to write ssh_config to %s: %w", sshConfigFile, err)
	}

	binDir := filepath.Join(configDir, "bin")

	err = os.MkdirAll(binDir, 0o750)
	if err != nil {
		return fmt.Errorf("failed to create bin vscode directory: %w", err)
	}
	log.Debug().Str("Agent", cfg.VsCode.Target).Str("BinDir", binDir).Msg("Generated bin directory")
	var suffix string
	if strings.HasSuffix(os.Args[0], ".exe") {
		suffix = ".exe"
	}
	sshPath := filepath.Join(binDir, "ssh"+suffix)

	executable, err := os.Executable()
	if err != nil {
		log.Error().Err(err).Str("Path", sshPath).Msg("Failed to get executable")

		return err
	}

	srcAbsBin, err := filepath.Abs(executable)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Path", os.Args[0]).Msg("Failed to get absolute path")

		return err
	}
	log.Trace().Str("Path", srcAbsBin).Msg("Generated absolute path")
	err = utils.CreateOrReplaceFileSymlink(srcAbsBin, sshPath)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Target", srcAbsBin).Str("Link", sshPath).Msg("Failed to create or replace ssh file")

		return fmt.Errorf("failed to create ssh symlink: %w", err)
	}
	log.Debug().Str("Target", srcAbsBin).Str("Link", sshPath).Msg("SSH symlink created")

	scpPath := filepath.Join(binDir, "scp"+suffix)
	err = utils.CreateOrReplaceFileSymlink(srcAbsBin, scpPath)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Target", srcAbsBin).Str("Link", scpPath).Msg("Failed to create or replace scp file")

		return fmt.Errorf("failed to create scp symlink: %w", err)
	}
	log.Debug().Str("Agent", cfg.VsCode.Target).Str("Target", srcAbsBin).Str("Link", scpPath).Msg("SCP symlink created")

	agent, err := clientAPI.GetAgentByName(cfg.VsCode.Target)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Target", cfg.VsCode.Target).Msg("Failed to get agent")

		return err
	}
	vsCodeSettings := GenVSCodeSettings(sshConfigFile, agent, sshPath)

	log.Debug().Str("Agent", cfg.VsCode.Target).Str("SSH config Path", vsCodeSettings.RemoteSSHConfigFile).Str("SSH bin Path", vsCodeSettings.RemoteSSHPath).Str("Remote install path", vsCodeSettings.InstallPath).Msg("Generate vscode config")

	vsCodeSettingsJSON, err := json.MarshalIndent(vsCodeSettings, "", "  ")
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("VsCode settings", fmt.Sprintf("%+v", vsCodeSettings)).Msg("Failed to marshal vscode settings")

		return err
	}

	settingsPath := filepath.Join(userVsCode, "settings.json")
	err = utils.WriteToFile(string(vsCodeSettingsJSON), settingsPath)
	if err != nil {
		log.Error().Err(err).Str("Agent", cfg.VsCode.Target).Str("Path", settingsPath).Msg("Failed to write vscode settings to file")

		return err
	}
	log.Trace().Str("Agent", cfg.VsCode.Target).Str("Path", settingsPath).Msg("VSCode settings written")

	//nolint:gosec
	cmd := exec.Command("code", "--user-data-dir", configDir, "--remote", "ssh-remote+"+cfg.VsCode.Target, cfg.VsCode.RemotePath)

	cmd.Env = os.Environ()
	if agent.HasStaticPassword && cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Error().Err(err).Str("Agent", agent.Name).Msg("Failed to prompt for static password")

			return err
		}
		cmd.Env = append(cmd.Env, prefixEnv("PASSWORD", cfg.PrivatePassword))
	}
	if cfg.ConfigFile != "" {
		cmd.Env = append(cmd.Env, prefixEnv("CONFIG_FILE", cfg.ConfigFile))
	}
	cwd, err := os.Getwd()
	if err == nil {
		cmd.Dir = cwd
	}

	log.Warn().Str("Path", vsCodeSettings.InstallPath).Msgf("Remember to delete the VSCode server temporary directory: %s", vsCodeSettings.InstallPath)
	log.Debug().Str("Agent", cfg.VsCode.Target).Str("Remote Path", cfg.VsCode.RemotePath).Str("Command", strings.Join(cmd.Args, " ")).Msg("Running vscode command")
	_, err = cmd.Output()
	if err != nil {
		log.Error().Err(err).Msg("Failed to run vscode command")

		return err
	}

	return nil
}

func GetConfigDir() string {
	dir, err := os.UserCacheDir()
	if err != nil {
		dir, err = os.Getwd()
		if err != nil {
			dir = os.TempDir()
		}
	}
	log.Debug().Str("dir", dir).Msg("Using current directory")
	configDir := filepath.Join(dir, "tealc-vscode")

	return configDir
}

func CheckVsCode() error {
	vscodeCmd := "code"
	_, err := exec.LookPath(vscodeCmd)
	if err != nil {
		return errors.New("vscode (code) not found in $PATH")
	}
	cmd := exec.Command(vscodeCmd, "--help")

	out, err := cmd.Output()
	if err != nil {
		return errors.New("vscode is not installed")
	}
	outStr := string(out)
	if strings.HasPrefix(outStr, "Visual Studio Code") {
		return nil
	}

	return fmt.Errorf("vscode is not installed, expected \"Visual Studio Code*\", got \"%s\"", strings.Split(outStr, "\n")[0])
}

type VsCodeSettings struct {
	RemoteSSHConfigFile            string            `json:"remote.SSH.configFile"`
	RemoteSSHPath                  string            `json:"remote.SSH.path"`
	RemoteSSHEnableX11Forwarding   bool              `json:"remote.SSH.enableX11Forwarding"`
	RemoteSSHExperimentalChat      bool              `json:"remote.SSH.experimental.chat"`
	RemoteSSHLocalServerDownload   string            `json:"remote.SSH.localServerDownload"`
	RemoteSSHEnableAgentForwarding bool              `json:"remote.SSH.enableAgentForwarding"`
	RemoteSSHServerInstallPath     map[string]string `json:"remote.SSH.serverInstallPath"`
	InstallPath                    string            `json:"-"`
}

func GenVSCodeSettings(sshConfigFile string, agent types.Agent, binPath string) VsCodeSettings {
	var sep string
	if runtime.GOOS == "windows" {
		sep = "\\"
	} else {
		sep = "/"
	}
	targetPath := fmt.Sprintf("%s%s%s", agent.Path, sep, vscode.VSCode)
	settings := VsCodeSettings{
		RemoteSSHConfigFile:            sshConfigFile,
		RemoteSSHPath:                  binPath,
		RemoteSSHEnableX11Forwarding:   false,
		RemoteSSHExperimentalChat:      false,
		RemoteSSHLocalServerDownload:   "always",
		RemoteSSHEnableAgentForwarding: false,
		RemoteSSHServerInstallPath: map[string]string{
			agent.Name: targetPath,
		},
		InstallPath: targetPath,
	}

	return settings
}

func GenSSHConfig(target string) string {
	split := strings.Split(target, "@")
	if len(split) == 1 {
		return fmt.Sprintf(`Host %s
  Hostname %s
`, target, target)
	}

	return fmt.Sprintf(`Host %s
  Hostname %s
  User %s
`, split[1], split[1], split[0])
}
