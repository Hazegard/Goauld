package main

import (
	"Goauld/client/api"
	commonCmd "Goauld/common/cmd"
	"Goauld/common/log"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/mattn/go-isatty"
)

// Rclone wraps the rclone command to copy files between the host and the agent.
type Rclone struct {
	Target    string   `kong:"-"` // internal, not shown in help
	Log       bool     `default:"${_rclone_log}" name:"log" yaml:"log" optional:"" help:"Record the SSH session to a log file."`
	Print     bool     `default:"${_rclone_print}" name:"print" yaml:"print" negatable:"" optional:"" help:"Print the generated rclone command instead of executing it."`
	AgentPath string   `arg:"" name:"agent" yaml:"agent" help:"AgentPath with paths that should be mounted."`
	LocalPath string   `arg:"" name:"local-path" yaml:"local-path" help:"Local path."`
	Args      []string `arg:"" name:"args" yaml:"args" help:"Paths to synchronize using rclone." passthrough:"" optional:""`
}

func (r *Rclone) Run(clientAPI *api.API, cfg ClientConfig) error {

	if len(commonCmd.CheckCommands([]string{"rclone"})) > 0 {
		log.Error().Str("Command", "rclone").Msg("Command not found")
		return errors.New("command not found: rclone")
	}
	log.Warn().Msgf("Rclone is running")
	ok, target := ExtractRemote(r.AgentPath)
	if !ok {
		return fmt.Errorf("could not extract remote agent: %s", r.AgentPath)
	}
	log.Debug().Msgf("Target: %s", target)
	r.Target = target
	cfg.Rclone.Target = r.Target

	log.Warn().Msgf("Rclone target: %s", r.Target)
	agent, err := clientAPI.GetAgentByName(target)
	if err != nil {
		return err
	}
	if !agent.Connected {
		return fmt.Errorf("unable to connect, agent %s (%s) not connected", agent.Name, agent.ID)
	}

	cfg.SSH = SSH{
		Target:            target,
		Socks:             false,
		HTTP:              false,
		HTTPMITM:          false,
		LocalSocksPort:    0,
		LocalHTTPPort:     0,
		LocalHTTPMITMPort: 0,
		SSH:               true,
		Print:             true,
		Proxy:             false,
		SSHOpts:           nil,
		SSHConfFile:       "",
		SSHArgs:           nil,
	}

	exePath, err := getPath()
	if err != nil {
		return err
	}
	log.Debug().Msgf("ExePath: %s", exePath)

	if cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Warn().Err(err).Msg("error while retrieving password from command line, ignoring...")
		}
	}

	proxyCmd := cfg.SSH.buildCommand(cfg, agent, exePath)
	//proxyCmdEnv := proxyCmd.InlineEnv()

	mountOption := "mount"
	if runtime.GOOS == "darwin" {
		mountOption = "nfsmount"
	}

	rcloneArgs := []string{
		"--sftp-ssh",
		strings.ReplaceAll(proxyCmd.String(), "'", "\""),
		mountOption,
		strings.ReplaceAll(r.AgentPath, target, ":sftp"),
		r.LocalPath,
	}

	rcloneArgs = append(rcloneArgs, r.Args...)
	rcloneCommand := Command{
		Executable: "rclone",
		Args:       rcloneArgs,
		Env:        proxyCmd.Env,
		Agent:      agent,
	}

	if cfg.Rclone.Print {
		//nolint:forbidigo
		fmt.Println(rcloneCommand.StringShell())

		return nil
	}
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	return rcloneCommand.Execute(cfg, target, isTerminal, r.Log)
}
