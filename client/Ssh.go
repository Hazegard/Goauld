package main

import (
	"Goauld/client/api"
	"Goauld/client/common"
	"Goauld/client/types"
	"Goauld/common/log"
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/aymanbagabas/go-pty"
)

type Ssh struct {
	Target         string   `arg:"" help:"The target agent." optional:""`
	Socks          bool     `default:"${_ssh_socks}" name:"socks" yaml:"socks" negatable:""  optional:"" help:"Forward the SOCKS ports on the local host."`
	Http           bool     `default:"${_ssh_http}" name:"http" yaml:"http" negatable:""  optional:"" help:"Forward the HTTP proxy ports on the local host."`
	LocalSocksPort int      `default:"${_ssh_local_socks_port}" name:"socks-port" yaml:"socks-port" optional:"" help:"Local port to bind the SOCKS to."`
	LocalHttpPort  int      `default:"${_ssh_local_http_port}" name:"http-port" yaml:"http-port" optional:"" help:"Local port to bind the SOCKS to."`
	Ssh            bool     `default:"${_ssh_ssh}" name:"ssh" yaml:"ssh" negatable:""  optional:"" help:"Connect to the agent SSHD service."`
	Print          bool     `default:"${_ssh_print}" name:"print" yaml:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Proxy          bool     `default:"${_ssh_proxy}" name:"proxy" yaml:"proxy" optional:"" help:"Enable direct STDIN/STDOUT connections to Allow to use proxycommand."`
	SshOpts        []string `short:"o"`
	SshConfFile    string   `short:"F"`
	SshArgs        []string `arg:"" passthrough:"" optional:"" help:"Additional args directly passed to the SSH command."`
}

type Command struct {
	Executable string
	Args       []string
	Env        []string
}

// InlineEnv modify the command to use the env binary to load the environment variables
func (c *Command) InlineEnv() *Command {
	args := c.Env
	args = append(args, c.Executable)
	args = append(args, c.Args...)
	c.Args = args
	c.Executable = "env"
	c.Env = []string{}
	return c
}

// String returns the command as a string
func (c *Command) String() string {
	return fmt.Sprintf("%s %s", c.Executable, strings.Join(c.Args, " "))
}

// StringShell returns the command as a string, each parameter escaped by quotes
func (c *Command) StringShell() string {
	args := []string{}
	for _, arg := range c.Args {
		args = append(args, fmt.Sprintf("'%s'", arg))
	}
	return fmt.Sprintf("%s %s", c.Executable, strings.Join(args, " "))
}

func (c *Command) Execute(cfg ClientConfig, target string, inPty bool) error {
	var err error
	for attempt := 0; attempt <= 3; attempt++ {
		hasFailed := false
		err, hasFailed = c.execute(cfg, inPty)
		if !hasFailed {
			return err
		}
		if attempt < 3 {
			err = cfg.Prompt(target)
			if err != nil {
				log.Warn().Err(err).Msg("error while retrieving password from command line, ignoring...")
				break
			}
			c.UpdatePwd(cfg.PrivatePassword)
		}
	}
	return err
}

func (c *Command) UpdatePwd(newPwd string) {
	for i := range c.Env {
		if strings.HasPrefix(c.Env[i], prefixEnv("PASSWORD", "")) {
			c.Env[i] = prefixEnv("PASSWORD", newPwd)
			return
		}
	}
	c.Env = append([]string{prefixEnv("PASSWORD", newPwd)}, c.Env...)
}

// Execute executes the command and adds the environment variables if needed
func (c *Command) execute(cfg ClientConfig, inPty bool) (error, bool) {
	var err error
	var stdoutPipe io.ReadCloser
	var stderrPipe io.ReadCloser

	var run func() error
	var wait func() error
	if inPty {
		var ptyFile pty.Pty
		ptyFile, err = pty.New()
		if err != nil {
			return err, false
		}
		cmd := ptyFile.Command(c.Executable, c.Args...)
		if len(c.Env) > 0 {
			cmd.Env = append(os.Environ(), c.Env...)
		}

		//nolint:errcheck
		defer ptyFile.Close()

		// Capture stdout and stderr from the PTY
		stdoutPipe = ptyFile
		stderrPipe = ptyFile
		run = func() error {
			return cmd.Start()
		}
		wait = func() error {
			return cmd.Wait()
		}
	} else {
		cmd := exec.Command(c.Executable, c.Args...)
		if len(c.Env) > 0 {
			cmd.Env = append(os.Environ(), c.Env...)
		}

		cmd.Stdin = os.Stdin
		stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return err, false
		}
		stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return err, false
		}
		run = func() error {
			return cmd.Start()
		}
		wait = func() error {
			return cmd.Wait()
		}

	}

	// Tee stdout
	prOut, pwOut := io.Pipe()
	teeOut := io.TeeReader(stdoutPipe, pwOut)

	// Tee stderr
	prErr, pwErr := io.Pipe()
	teeErr := io.TeeReader(stderrPipe, pwErr)

	// Let output go to terminal
	go func() {
		//nolint:errcheck
		defer pwOut.Close()
		_, _ = io.Copy(os.Stdout, teeOut)
	}()

	go func() {
		//nolint:errcheck
		defer pwErr.Close()
		_, _ = io.Copy(os.Stderr, teeErr)
	}()

	defer func() {
		_ = pwOut.Close()
		_ = pwErr.Close()
		_ = prErr.Close()
		_ = prOut.Close()
		_ = stderrPipe.Close()
		_ = stdoutPipe.Close()
	}()

	// Scanner for stderr to detect failure
	wg := sync.WaitGroup{}
	var hasAuthFailed atomic.Bool

	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(prErr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "Permission denied, please try again.") {
				hasAuthFailed.Store(true)
				break
			}
		}
		_, _ = io.Copy(io.Discard, prErr)
	}()

	// Scanner for stdout (if you want to inspect success messages etc.)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(prOut)
		for scanner.Scan() {
			if !hasAuthFailed.Load() && cfg.SavePassword {
				err = cfg.UpdatePassConfigFile()
				if err != nil {
					log.Warn().Err(err).Msg("Failed to update config file")
				}
				break
			}
		}
		_, _ = io.Copy(io.Discard, prOut)
	}()

	err = run()
	if err != nil {
		return err, hasAuthFailed.Load()
	}

	err = wait()

	wg.Wait()
	return err, hasAuthFailed.Load()
}

// Run execute the ssh subcommand
func (e *Ssh) Run(api *api.API, cfg ClientConfig) error {

	for i := range e.SshArgs {
		if cfg.Ssh.SshArgs[i] == "-F" {
			cfg.Ssh.SshConfFile = cfg.Ssh.SshArgs[i+1]
		}
	}
	if cfg.Socks.Target != "" {
		// we are in socks mode, so apply the socks option to the ssh
		cfg.Ssh = Ssh{
			Target:         cfg.Socks.Target,
			Socks:          cfg.Socks.Socks,
			Http:           cfg.Ssh.Http,
			LocalSocksPort: cfg.Socks.LocalSocksPort,
			LocalHttpPort:  cfg.Socks.LocalHttpPort,
			Ssh:            false,
			Print:          cfg.Ssh.Print,
			Proxy:          false,
			SshArgs:        cfg.Socks.SshArgs,
			SshConfFile:    cfg.ConfigFile,
		}
	}
	if e.Proxy {
		e.Socks = false
		e.Http = false
	}
	return e.Execute(api, cfg)
}

// Execute start the ssh
func (e *Ssh) Execute(api *api.API, cfg ClientConfig) error {
	if len(e.SshArgs) == 1 && e.SshArgs[0] == "" {
		e.SshArgs = []string{}
	}
	agent, err := api.GetAgentByName(cfg.Ssh.Target)
	if err != nil {
		log.Warn().Err(err).Str("target", cfg.Ssh.Target).Msg("Failed to get agent")
		cfg.Ssh.Target, err = GetFromSSHConfig(cfg.Ssh.SshConfFile, cfg.Ssh.Target)

		log.Debug().Str("Target", cfg.Ssh.Target).Msg("Trying using ssh_config file")

		agent, err = api.GetAgentByName(cfg.Ssh.Target)
		if err != nil {
			return fmt.Errorf("failed to get agent by name (%s): %s", cfg.Ssh.Target, err)
		}
	}
	if !agent.Connected {
		return fmt.Errorf("unable to connect, agent %s (%s) not connected", agent.Name, agent.Id)
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

	cmd := e.buildCommand(cfg, agent, exePath)
	if e.Print {
		fmt.Println(cmd.InlineEnv().String())
		return nil
	}
	if e.Proxy {
		return cmd.Execute(cfg, agent.Name, false)
	}

	err = cmd.Execute(cfg, agent.Name, false)
	if err != nil {
		var exitError *exec.ExitError
		ok := errors.As(err, &exitError)
		if ok {
			// Get the exit status
			exitStatus := exitError.ExitCode()
			if exitStatus == 255 || exitStatus == 4294967295 {
				return nil
			}
			return err
		} else {
			return err
		}
	}
	return nil
}

// buildCommand build the ssh command
func (e *Ssh) buildCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	if e.Proxy {
		return e.buildTunnelSshCommand(cfg, agent, exePath)
	}
	if e.Ssh {
		return e.buildOuterSshCommand(cfg, agent, exePath)
	}
	cmd := e.buildTunnelSshCommand(cfg, agent, exePath)
	cmd.Args = append(cmd.Args, "-N")
	return cmd
}

// buildOuterSshCommand build the outer SSH command. This SSH command will be executed in second
// through the ProxyCommand
func (e *Ssh) buildOuterSshCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	innerCmd := e.buildTunnelSshCommand(cfg, agent, exePath)
	cmd := Command{}
	cmd.Executable = "ssh"
	cmd.Args = buildAllSshOptions(cfg)
	cmd.Env = buildEnvironments(cfg, "agent", exePath, cfg.Ssh.Target)
	if e.Print {
		for i := range cmd.Env {
			cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
		}
	}
	// We display the proxycommand inside single quotes to allow users to copy and paste the command
	sep := ""
	if e.Print {
		sep = "'"
	}
	proxyCmd := ""
	if e.Print {
		proxyCmd = fmt.Sprintf("-oProxyCommand=%s%s%s", sep, innerCmd.InlineEnv().String(), sep)
	} else {
		proxyCmd = fmt.Sprintf("-oProxyCommand=%s%s%s", sep, innerCmd.String(), sep)
	}
	cmd.Args = append(cmd.Args, proxyCmd)
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, agent.Id))
	cmd.Args = append(cmd.Args, cfg.Ssh.SshArgs...)
	return cmd
}

// buildTunnelSshCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (i.e.: it will be executed first)
func (e *Ssh) buildTunnelSshCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	cmd := Command{
		Executable: "ssh",
	}
	cmd.Env = buildEnvironments(cfg, "otp", exePath, cfg.Ssh.Target)
	for i := range cmd.Env {
		cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
	}
	cmd.Args = buildInnerSshOptions(cfg)
	cmd.Args = append(cmd.Args, fmt.Sprintf("-p%s", cfg.GetSshdPort()))
	if e.Ssh || e.Proxy {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-W127.0.0.1:%s", agent.GetSSHPort()))
	}
	if e.Socks && agent.GetSocksPort() != "0" && agent.GetSocksPort() != ":" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-L%d:127.0.0.1:%s", cfg.Ssh.LocalSocksPort, agent.GetSocksPort()))
	}
	if e.Http && agent.GetHttpPort() != "0" && agent.GetHttpPort() != "/" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-L%d:127.0.0.1:%s", cfg.Ssh.LocalHttpPort, agent.GetHttpPort()))
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost()))
	return cmd
}

// buildEnvironments returns the environment variables required to access the agents
func buildEnvironments(cfg ClientConfig, typePass string, exePath string, target string) []string {
	envs := []string{
		"SSH_ASKPASS_REQUIRE=force",
		"SSH_ASKPASS=" + exePath,
		prefixEnv("SERVER", cfg.Server),
		prefixEnv("AGENT", target),
		prefixEnv("TYPE", typePass),
		prefixEnv("CONFIG_FILE", cfg.ConfigFile),
	}
	if cfg.PrivatePassword != "" {
		envs = append(envs, prefixEnv("PASSWORD", cfg.PrivatePassword))
	}
	if cfg.IsFlagInCommandLine("--access-token", "") {
		envs = append(envs, prefixEnv("ACCESS_TOKEN", cfg.AccessToken))
	}
	return envs
}

// buildAllSshOptions returns the ssh options that are in common to the inner and the outer ssh
func buildAllSshOptions(cfg ClientConfig) []string {
	options := []string{
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-oPubkeyAuthentication=no",
		"-oPreferredAuthentications=password",
		"-oLogLevel=ERROR",
		"-oExitOnForwardFailure=no",
	}

	if cfg.Verbose > 0 {
		options = append(options, fmt.Sprintf("-%s", strings.Repeat("v", cfg.Verbose)))
	}
	return options
}

// buildSshOptions returns the SSH options required to access the agents
func buildInnerSshOptions(cfg ClientConfig) []string {
	options := []string{
		"-oClearAllForwardings=no",
		//"-vv",
	}
	return append(options, buildAllSshOptions(cfg)...)
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

func ExecuteSystemSSH(args ...string) {
	cmd := exec.Command("ssh", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Env = os.Environ()
	err := cmd.Run()
	if err != nil {
		log.Error().Err(err).Msg("error running ssh")
		return
	}
}
