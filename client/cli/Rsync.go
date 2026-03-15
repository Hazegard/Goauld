package cli

import (
	"Goauld/client/api"
	commonCmd "Goauld/common/cmd"
	"Goauld/common/log"
	"errors"
	"fmt"
	"os"

	"github.com/mattn/go-isatty"
)

// Rsync wraps the rsync command to copy files between the host and the agent.
type Rsync struct {
	Target string   `kong:"-"` // internal, not shown in help
	Log    bool     `default:"${_rsync_log}" name:"log" yaml:"log" optional:"" help:"Record the SSH session to a log file."`
	Print  bool     `default:"${_rsync_print}" name:"print" yaml:"print" negatable:"" optional:"" help:"Print the generated rsync command instead of executing it."`
	Args   []string `arg:"" name:"paths" yaml:"paths" help:"Paths to synchronize using rsync." passthrough:""`
}

func getRawRsyncArgs() []string {
	isRsyncParam := false
	params := []string{"-r", "-v", "-P"}
	for _, arg := range os.Args {
		if arg == "--log" || arg == "--print" {
			continue
		}
		if isRsyncParam {
			params = append(params, arg)

			continue
		}
		if arg == "rsync" {
			isRsyncParam = true
		}
	}

	return params
}

func (r *Rsync) Run(clientAPI *api.API, cfg ClientConfig) error {
	if len(commonCmd.CheckCommands([]string{"rsync"})) > 0 {
		log.Error().Str("Command", "rsync").Msg("Command not found")

		return errors.New("command not found: rsync")
	}
	r.Args = getRawRsyncArgs()

	target, err := extractTarget(r.Args)
	if err != nil {
		return err
	}
	r.Target = target
	cfg.Rsync.Target = r.Target

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
	if cfg.ShouldPrompt(agent) {
		err = cfg.Prompt(agent.Name)
		if err != nil {
			log.Warn().Err(err).Msg("error while retrieving password from command line, ignoring...")
		}
	}

	proxyCmd := cfg.SSH.buildCommand(cfg, agent, exePath)
	proxyCmdEnv := proxyCmd.InlineEnv()

	rsyncArgs := []string{"-e", proxyCmdEnv.String()}
	rsyncArgs = append(rsyncArgs, r.Args...)
	rsyncCommand := Command{
		Executable: "rsync",
		Args:       rsyncArgs,
		Env:        nil,
		Agent:      agent,
	}

	if cfg.Rsync.Print {
		//nolint:forbidigo
		fmt.Println(rsyncCommand.StringShell())

		return nil
	}
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	return rsyncCommand.Execute(cfg, target, isTerminal, r.Log)
}
