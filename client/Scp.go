package main

import (
	"Goauld/common/log"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"Goauld/client/api"
	"Goauld/client/types"

	"github.com/mattn/go-isatty"
)

// Scp wraps the scp command to copy files between the host and the agent.
type Scp struct {
	Target string `kong:"-"` // internal, not shown in help
	Print  bool   `default:"${_scp_print}" name:"print" yaml:"print" optional:"" help:"Print the generated SCP command instead of executing it."`

	SSHOpts     []string `short:"o" name:"ssh-opts" yaml:"ssh-opts" help:"Additional SSH options (equivalent to '-o')."`
	SSHConfFile string   `short:"F" name:"ssh-config-file" yaml:"ssh-config-file" help:"Path to an SSH configuration file to use."`
	Paths       []string `arg:"" name:"paths" yaml:"paths" help:"Paths to copy using SCP." passthrough:""`
}

// Run execute the scp command.
func (s *Scp) Run(clientAPI *api.API, cfg ClientConfig) error {
	return s.Execute(clientAPI, cfg)
}

func extractTarget(paths []string) (string, error) {
	for _, p := range paths {
		isRemote, target := ExtractRemote(p)
		if isRemote {
			return target, nil
		}
	}

	// isRemote, target := ExtractRemote(s.Source)
	// if isRemote {
	// return target, nil
	// }

	// isRemote, target = ExtractRemote(s.Destination)
	// if isRemote {
	// return target, nil
	// }
	return "", fmt.Errorf("SCP target not found in [%s]", strings.Join(paths, ", "))
}

// GetTarget parses the input and fetches the target agent, whether it is in the source or destination of the scp command.
func (s *Scp) GetTarget() (string, error) {
	return extractTarget(s.Paths)
}

// ExtractRemote tries to find the remote agent, whether located in the source or in the destination of the command.
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

// Execute start the ssh.
func (s *Scp) Execute(clientAPI *api.API, cfg ClientConfig) error {
	target, err := s.GetTarget()
	if err != nil {
		return err
	}
	s.Target = target
	cfg.SCP.Target = target
	agent, err := clientAPI.GetAgentByName(cfg.SCP.Target)
	if err != nil {
		return err
	}
	if !agent.Connected {
		return fmt.Errorf("unable to connect, agent %s (%s) not connected", agent.Name, agent.ID)
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

	cmd := s.buildScpCommand(cfg, agent, exePath)
	if s.Print {
		//nolint:forbidigo
		fmt.Println(cmd.InlineEnv().StringShell())

		return nil
	}
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())

	return cmd.Execute(cfg, agent.Name, isTerminal)
}

// buildScpCommand build the outer SSH command. This SSH command will be executed in second
// through the ProxyCommand.
func (s *Scp) buildScpCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	proxyCmd := s.buildTunnelSSHCommand(cfg, agent, exePath)

	cmd := Command{
		Agent: agent,
	}
	cmd.Executable = "scp"
	cmd.Args = append([]string{"-r"}, buildAllSSHOptions(cfg)...)
	cmd.Env = buildEnvironments(cfg, "agent", exePath, cfg.SCP.Target)
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
	// cmd.Args = append(cmd.Args, cfg.SCP.ScpArgs...)
	for _, opt := range s.SSHOpts {
		cmd.Args = append(cmd.Args, "-o"+opt)
	}
	if s.SSHConfFile != "" {
		cmd.Args = append(cmd.Args, "-F", s.SSHConfFile)
	}
	for i := range s.Paths {
		s.Paths[i] = strings.ReplaceAll(s.Paths[i], agent.Name, fmt.Sprintf("%s@%s", agent.Name, agent.ID))
	}
	cmd.Args = append(cmd.Args, s.Paths...)
	//	cmd.Args = append(cmd.Args, s.Destination)
	// cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, agent.ID))
	return cmd
}

// buildTunnelSSHCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (i.e.: it will be executed first).
func (s *Scp) buildTunnelSSHCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	cmd := Command{
		Executable: "ssh",
		Agent:      agent,
	}
	cmd.Env = buildEnvironments(cfg, "otp", exePath, cfg.SCP.Target)
	for i := range cmd.Env {
		cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
	}
	cmd.Args = buildInnerSSHOptions(cfg)
	cmd.Args = append(cmd.Args, "-p"+cfg.GetSshdPort())
	cmd.Args = append(cmd.Args, "-W127.0.0.1:"+agent.GetSSHPort())

	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost()))

	return cmd
}

func ExecSCp() error {
	cmd := exec.Command("scp")
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()

	return cmd.Run()
}
