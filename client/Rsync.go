package main

import (
	"Goauld/client/api"
	"fmt"
	"os"
	"strings"

	"github.com/mattn/go-isatty"
)

// Rsync wraps the rsync command to copy files between the host and the agent.
type Rsync struct {
	Target string `kong:"-"`
	Print  bool   `default:"${_scp_print}" name:"print" yaml:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	// Source      string   `default:"${_scp_source}" arg:"" name:"source" help:"Origin copy."`
	// Destination string   `default:"${_scp_destination}" arg:"" name:"destination" yaml:"destination" help:"Destination to copy."`
	// RsyncOpts []string `short:"o"`
	Args []string `arg:"" name:"paths" help:"List of paths to scp" passthrough:""`
}

func getRawRsyncArgs() []string {
	isRsyncParam := false
	params := []string{}
	for _, arg := range os.Args {
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
	r.Args = getRawRsyncArgs()

	target, err := extractTarget(r.Args)
	if err != nil {
		return err
	}
	r.Target = target
	cfg.Rsync.Target = r.Target

	for i := range r.Args {
		r.Args[i] = strings.TrimPrefix(r.Args[i], target)
	}

	agent, err := clientAPI.GetAgentByName(target)
	if err != nil {
		return err
	}
	if !agent.Connected {
		return fmt.Errorf("unable to connect, agent %s (%s) not connected", agent.Name, agent.ID)
	}

	cfg.SSH = SSH{
		Target:         target,
		Socks:          false,
		HTTP:           false,
		LocalSocksPort: 0,
		LocalHTTPPort:  0,
		SSH:            true,
		Print:          true,
		Proxy:          false,
		SSHOpts:        nil,
		SSHConfFile:    "",
		SSHArgs:        nil,
	}

	exePath, err := getPath()
	if err != nil {
		return err
	}

	proxyCmd := cfg.SSH.buildCommand(cfg, agent, exePath)
	proxyCmdEnv := proxyCmd.InlineEnv()

	rsyncArgs := []string{"-e", proxyCmdEnv.String()}
	rsyncArgs = append(rsyncArgs, r.Args...)
	rsyncCommand := Command{
		Executable: "rsync",
		Args:       rsyncArgs,
		Env:        nil,
	}
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	return rsyncCommand.Execute(cfg, target, isTerminal)
}
