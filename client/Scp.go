package main

import (
	"fmt"
	"regexp"
	"strings"

	"Goauld/client/api"
	"Goauld/client/types"
)

// Scp wraps the scp command to copy files between the host and the agent
type Scp struct {
	Target      string
	Print       bool     `default:"${_scp_print}" name:"print" yaml:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Source      string   `default:"${_scp_source}" arg:"" name:"source" help:"Origin copy."`
	Destination string   `default:"${_scp_destination}" arg:"" name:"destination" yaml:"destination" help:"Destination to copy."`
	ScpArgs     []string `arg:"" passthrough:"" optional:""`
}

// Run execute the scp command
func (s *Scp) Run(api *api.API, cfg ClientConfig) error {
	return s.Execute(api, cfg)
}

// GetTarget parses the input and fetches the target agent, whether it is in the source or destination of the scp command
func (s *Scp) GetTarget() (string, error) {

	isRemote, target := ExtractRemote(s.Source)
	if isRemote {
		return target, nil
	}

	isRemote, target = ExtractRemote(s.Destination)
	if isRemote {
		return target, nil
	}
	return "", fmt.Errorf("SCP target not found in %s or %s", s.Source, s.Destination)
}

// ExtractRemote tries to find the remote agent, whether located in the source or in the destination of the command
func ExtractRemote(s string) (bool, string) {
	parts := strings.Split(s, ":")
	if len(parts) == 1 {
		return false, ""
	}
	part := parts[0]
	remaining := strings.Join(parts[1:], ":")
	windriveRegex := regexp.MustCompile(`^[A-Za-z]:[\\/]`)
	if windriveRegex.MatchString(s) {
		return false, ""
	}
	if windriveRegex.MatchString(remaining) {
		return true, part
	}
	return true, part
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

	return cmd.Execute(cfg, agent.Name)
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
	// We display the proxycommand inside single quotes to allow users to copy and paste the command
	sep := ""
	if s.Print {
		sep = "'"
	}
	proxyScpCmd := fmt.Sprintf("-oProxyCommand=%s%s%s", sep, proxyCmd.String(), sep)
	cmd.Args = append(cmd.Args, proxyScpCmd)
	cmd.Args = append(cmd.Args, cfg.Scp.ScpArgs...)
	cmd.Args = append(cmd.Args, s.Source)
	cmd.Args = append(cmd.Args, s.Destination)
	// cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, agent.Id))
	return cmd
}

// buildTunnelSshCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (i.e.: it will be executed first)
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
