package main

import (
	"fmt"
	"strings"

	"Goauld/client/api"
	"Goauld/client/types"
)

type Scp struct {
	Target      string
	Print       bool     `default:"${_exec_print}" name:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Source      string   `arg:"" name:"source" help:"Origin copy."`
	Destination string   `arg:"" name:"destination" help:"Destination to copy."`
	ScpArgs     []string `arg:"" passthrough:"" optional:""`
}

func (s *Scp) Run(api *api.API, cfg ClientConfig) error {
	return s.Execute(api, cfg)
}

func (s *Scp) GetTarget() (string, error) {
	split := strings.Split(s.Source, ":")
	if len(split) == 2 {
		return split[0], nil
	}
	split = strings.Split(s.Destination, ":")
	if len(split) == 2 {
		return split[0], nil
	}
	return "", fmt.Errorf("SCP target not found in %s or %s", s.Source, s.Destination)
}

// Execute start the ssh
func (s *Scp) Execute(api *api.API, cfg ClientConfig) error {
	target, err := s.GetTarget()
	if err != nil {
		return err
	}
	s.Target = target
	cfg.Scp.Target = target
	agent, err := api.GetAgentByName(cfg.Scp.Target)
	if err != nil {
		return err
	}
	if !agent.Connected {
		return fmt.Errorf("unable to connect, agent %s (%s) not connected", agent.Name, agent.Id)
	}

	exePath, err := getPath()
	if err != nil {
		return err
	}

	cmd := s.buildScpCommand(cfg, agent, exePath)
	if s.Print {
		fmt.Println(cmd.InlineEnv().String())
		return nil
	}

	return cmd.Execute()
}

// buildScpCommand build the outer SSH command. This SSH command will be executed in second
// through the ProxyCommand
func (s *Scp) buildScpCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {

	proxyCmd := s.buildTunnelSshCommand(cfg, agent, exePath)

	cmd := Command{}
	cmd.Executable = "scp"
	cmd.Args = append([]string{"-r"}, buildAllSshOptions(cfg)...)
	cmd.Env = buildEnvironments(cfg, "agent", exePath, cfg.Scp.Target)
	if s.Print {
		for i := range cmd.Env {
			cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
		}
	}
	// We display the proxycommand inside single quotes in order to allow users to copy and paste the command
	sep := ""
	if s.Print {
		sep = "'"
	}
	proxyScpCmd := fmt.Sprintf("-oProxyCommand=%s%s%s", sep, proxyCmd.InlineEnv().String(), sep)
	cmd.Args = append(cmd.Args, proxyScpCmd)
	cmd.Args = append(cmd.Args, cfg.Scp.ScpArgs...)
	cmd.Args = append(cmd.Args, s.Source)
	cmd.Args = append(cmd.Args, s.Destination)
	// cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, agent.Id))
	return cmd
}

// buildTunnelSshCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (ie: it will be executed first)
func (s *Scp) buildTunnelSshCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	cmd := Command{
		Executable: "ssh",
	}
	cmd.Env = buildEnvironments(cfg, "otp", exePath, cfg.Scp.Target)
	for i := range cmd.Env {
		cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
	}
	cmd.Args = buildInnerSshOptions(cfg)
	cmd.Args = append(cmd.Args, fmt.Sprintf("-p%s", cfg.GetSshdPort()))
	cmd.Args = append(cmd.Args, fmt.Sprintf("-W127.0.0.1:%s", agent.GetSSHPort()))

	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost()))
	return cmd
}
