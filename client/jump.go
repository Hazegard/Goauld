package main

import (
	"fmt"
	"os"
)

type Jump struct {
	Agent string   `arg:"" name:"agent" yaml:"agent" help:"Agent name to retrieve password."`
	Print bool     `name:"print" yaml:"print" help:"Print this agent."`
	Scp   bool     `name:"scp" yaml:"scp" help:"Scp using the Jump."`
	Args  []string `arg:"" `
}

func (j *Jump) Run(cfg ClientConfig) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
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
