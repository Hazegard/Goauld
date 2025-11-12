package main

import (
	"Goauld/client/api"
	"fmt"
	"os"
	"strings"
)

type Jump struct {
	Agent string   `arg:"" name:"agent" yaml:"agent" help:"Target agent to use as a jump host for SSH connections."`
	Print bool     `name:"print" yaml:"print" help:"Print the generated SSH jump command instead of executing it."`
	Scp   bool     `name:"scp" yaml:"scp" help:"Use SCP through the jump host."`
	Log   bool     `default:"false" name:"log" yaml:"log" optional:"" help:"Record the SSH session to a log file."`
	Args  []string `arg:"" yaml:"args" passthrough:"" help:"Additional arguments passed to the SSH or SCP command."`
}

func (j *Jump) Run(clientAPI *api.API, cfg ClientConfig) error {
	agent, err := clientAPI.GetAgentByName(j.Agent)
	if err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	exe = strings.ReplaceAll(exe, ` `, `\ `)
	proxyCommand := fmt.Sprintf("%s ssh %s -W %%h:%%p", exe, j.Agent)
	args := []string{"-oProxycommand=" + proxyCommand}
	args = append(args, j.Args...)

	exeCmd := "ssh"
	if j.Scp {
		exeCmd = "scp"
	}
	cmd := Command{
		Executable: exeCmd,
		Args:       args,
		Env:        cfg.EnvVar(j.Agent),
		Log:        j.Log,
		Agent:      agent,
	}
	if j.Print {
		//nolint:forbidigo
		fmt.Println(cmd.String())

		return nil
	}

	return cmd.Execute(cfg, j.Agent, j.Scp)
}
