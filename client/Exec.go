package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"Goauld/client/api"
	"Goauld/client/types"
	"Goauld/common"
)

type Exec struct {
	Target         string `arg:""`
	Socks          bool   `default:"${_exec_socks}" name:"socks" negatable:""  optional:"" help:"Expose the agent SOCKS service."`
	LocalSocksPort int    `default:"${_local_socks_port}" name:"socksPort" optional:"" help:"Forwarded SOCKS Port."`
	Ssh            bool   `default:"${_exec_ssh}" name:"ssh" negatable:""  optional:"" help:"Connect to the agent SSHD service."`
	Print          bool   `default:"${_exec_print}" name:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Proxy          bool   `default:"${_exec_print}" name:"proxy" optional:"" help:"Allow to use proxycommand ."`
}

func (e *Exec) Run(api *api.API, cfg ClientConfig) error {
	return e.Execute(api, cfg)
}

// Execute start the ssh
func (e *Exec) Execute(api *api.API, cfg ClientConfig) error {
	agent, err := api.GetAgentByName(cfg.Exec.Target)
	if err != nil {
		return err
	}

	exePath, err := getPath()
	if err != nil {
		return err
	}

	cmd := e.buildCommand(cfg, agent, exePath)
	if e.Print {
		e.PrintCommand(cmd, cfg, exePath)
	}
	if e.Proxy {
		return e.ProxyCommand(cmd, cfg, exePath)
	}

	return e.ExecuteCommand(cmd, cfg, exePath)
}

func (e *Exec) ProxyCommand(cmd []string, cfg ClientConfig, exePath string) error {
	c := exec.Command("env", cmd...)
	c.Env = append(os.Environ(), buildEnvironments(cfg, "otp", exePath)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	err := c.Run()
	if err != nil {
		return err
	}

	return nil
}

func (e *Exec) ExecuteCommand(cmd []string, cfg ClientConfig, exePath string) error {
	c := exec.Command(cmd[0], cmd[1:]...)
	c.Env = append(os.Environ(), buildEnvironments(cfg, "agent", exePath)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	c.Stdin = os.Stdin
	err := c.Run()
	if err != nil {
		return err
	}

	return nil
}

func (e *Exec) PrintCommand(cmd []string, cfg ClientConfig, exePath string) {
	env := buildEnvironments(cfg, "agent", exePath)

	cmd = append(env, cmd...)

	for _, c := range cmd {
		fmt.Println(c)
	}
	fmt.Println()
	fmt.Println()
	fmt.Println()
	stringCmd := strings.Join(cmd, " ")
	fmt.Println(stringCmd)
}

func (e *Exec) buildCommand(cfg ClientConfig, agent types.Agent, exePath string) []string {
	if e.Proxy {
		return append(e.buildTunnelSshCommand(cfg, agent, exePath))
	}
	if e.Ssh {
		return e.buildOuterSshCommand(cfg, agent, exePath)
	}
	return e.buildTunnelSshCommand(cfg, agent, exePath)
}

// buildOuterSshCommand build the outer SSH command. This SSH command will be executed in second
// through the ProxyCommand
func (e *Exec) buildOuterSshCommand(cfg ClientConfig, agent types.Agent, exePath string) []string {
	var cmd []string
	cmd = append(cmd, "ssh")
	cmd = append(cmd, buildSshOptions("agent")...)
	sep := ""
	if e.Print {
		sep = "'"
	}
	tunnelCommand := append([]string{"env"}, buildEnvironments(cfg, "otp", exePath)...)
	tunnelSshCommand := append(tunnelCommand, e.buildTunnelSshCommand(cfg, agent, exePath)...)
	tunnelSsh := strings.Join(tunnelSshCommand, " ")
	cmd = append(cmd, fmt.Sprintf("-oProxyCommand=%s%s%s", sep, tunnelSsh, sep))
	cmd = append(cmd, fmt.Sprintf("%s@%s", agent.Name, agent.Id))
	return cmd
}

// buildTunnelSshCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (ie: it will be executed first)
func (e *Exec) buildTunnelSshCommand(cfg ClientConfig, agent types.Agent, exePath string) []string {
	var cmd []string
	cmd = append(cmd, "ssh")
	cmd = append(cmd, buildSshOptions("otp")...)
	if e.Ssh || e.Proxy {
		cmd = append(cmd, fmt.Sprintf("-W127.0.0.1:%s", agent.GetSSHPort()))
	}
	if e.Socks {
		cmd = append(cmd, fmt.Sprintf("-L%d:127.0.0.1:%s", cfg.Exec.LocalSocksPort, agent.GetSocksPort()))
	}
	cmd = append(cmd, fmt.Sprintf("%s@localhost", agent.Name))
	cmd = append(cmd, fmt.Sprintf("-p%s", cfg.GetSshdPort()))
	return cmd
}

// buildEnvironments returns the environment variables required to access the agents
func buildEnvironments(cfg ClientConfig, typePass string, exePath string) []string {
	return []string{
		"SSH_ASKPASS_REQUIRE=force",
		"SSH_ASKPASS=" + exePath,
		prefixEnv("SERVER", cfg.Server),
		prefixEnv("AGENT", cfg.Exec.Target),
		prefixEnv("TYPE", typePass),
	}
}

// buildSshOptions returns the SSH options required to access the agents
func buildSshOptions(typePass string) []string {
	options := []string{
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-oPubkeyAuthentication=no",
		"-oPreferredAuthentications=password",
		"-oClearAllForwardings=no",
		"-oBatchMode=yes",
		//	"-v",
	}
	return options
}

// prefixEnv adds the application name to the provided value and returns it
// as an environment variable
func prefixEnv(name string, value string) string {
	return fmt.Sprintf("%s_%s=%s", strings.ToUpper(common.APP_NAME), strings.ToUpper(name), value)
}

// getPath returns the path of the binary currently being executed
func getPath() (string, error) {
	// Get the executable's path
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	// Get the absolute path of the executable
	absPath, err := filepath.Abs(exePath)
	if err != nil {
		return "", err
	}

	// Resolve any symlinks, in case the executable is a symlink
	resolvedPath, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", err
	}

	return resolvedPath, nil
}
