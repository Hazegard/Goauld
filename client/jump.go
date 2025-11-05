package main

import (
	"fmt"
	"os"
	"strings"
)

type Jump struct {
	Agent string   `arg:"" name:"agent" yaml:"agent" help:"Target agent to use as a jump host for SSH connections."`
	Print bool     `name:"print" yaml:"print" help:"Print the generated SSH jump command instead of executing it."`
	Scp   bool     `name:"scp" yaml:"scp" help:"Use SCP through the jump host."`
	Args  []string `arg:"" passthrough:"" help:"Additional arguments passed to the SSH or SCP command."`
}

func (j *Jump) Run(cfg ClientConfig) error {
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
	}
	if j.Print {
		//nolint:forbidigo
		fmt.Println(cmd.String())

		return nil
	}

	return cmd.Execute(cfg, j.Agent, j.Scp)
}
