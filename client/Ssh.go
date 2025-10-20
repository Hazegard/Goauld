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

type SSH struct {
	Target         string   `arg:"" name:"agent" help:"The target agent." optional:""`
	Socks          bool     `default:"${_ssh_socks}" name:"socks" yaml:"socks" negatable:""  optional:"" help:"Forward the SOCKS ports on the local host."`
	HTTP           bool     `default:"${_ssh_http}" name:"http" yaml:"http" negatable:"" optional:"" help:"Forward the HTTP proxy ports on the local host."`
	LocalSocksPort int      `default:"${_ssh_local_socks_port}" name:"socks-port" yaml:"socks-port" optional:"" help:"Local port to bind the SOCKS to."`
	LocalHTTPPort  int      `default:"${_ssh_local_http_port}" name:"http-port" yaml:"http-port" optional:"" help:"Local port to bind the SOCKS to."`
	SSH            bool     `default:"${_ssh_ssh}" name:"ssh" yaml:"ssh" negatable:"" optional:"" help:"Connect to the agent SSHD service."`
	Print          bool     `default:"${_ssh_print}" name:"print" yaml:"print" negatable:""  optional:"" help:"Show the SSH command instead of executing it."`
	Proxy          bool     `default:"${_ssh_proxy}" name:"proxy" yaml:"proxy" optional:"" help:"Enable direct STDIN/STDOUT connections to Allow to use proxycommand."`
	SSHOpts        []string `short:"o"`
	SSHConfFile    string   `short:"F"`
	SSHArgs        []string `arg:"" passthrough:"" optional:"" help:"Additional args directly passed to the SSH command."`
}

type Command struct {
	Executable string
	Args       []string
	Env        []string
}

// InlineEnv modify the command to use the env binary to load the environment variables.
func (c *Command) InlineEnv() *Command {
	args := c.Env
	args = append(args, c.Executable)
	args = append(args, c.Args...)
	c.Args = args
	c.Executable = "env"
	c.Env = []string{}

	return c
}

// String returns the command as a string.
func (c *Command) String() string {
	return fmt.Sprintf("%s %s", c.Executable, strings.Join(c.Args, " "))
}

// StringShell returns the command as a string, each parameter escaped by quotes.
func (c *Command) StringShell() string {
	var args []string
	for _, arg := range c.Args {
		args = append(args, fmt.Sprintf("'%s'", arg))
	}

	return fmt.Sprintf("%s %s", c.Executable, strings.Join(args, " "))
}

func (c *Command) Execute(cfg ClientConfig, target string, inPty bool) error {
	var err error
	for attempt := range 4 {
		var hasFailed bool
		hasFailed, err = c.execute(cfg, inPty)
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

// Execute executes the command and adds the environment variables if needed.
func (c *Command) execute(cfg ClientConfig, inPty bool) (bool, error) {
	var err error
	var stdoutPipe io.ReadCloser
	var stderrPipe io.ReadCloser

	var run func() error
	var wait func() error
	if inPty {
		var ptyFile pty.Pty
		ptyFile, err = pty.New()
		if err != nil {
			return false, err
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
		//nolint:gosec
		cmd := exec.Command(c.Executable, c.Args...)
		if len(c.Env) > 0 {
			cmd.Env = append(os.Environ(), c.Env...)
		}

		cmd.Stdin = os.Stdin
		stdoutPipe, err = cmd.StdoutPipe()
		if err != nil {
			return false, err
		}
		stderrPipe, err = cmd.StderrPipe()
		if err != nil {
			return false, err
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
		return hasAuthFailed.Load(), err
	}

	err = wait()

	wg.Wait()

	return hasAuthFailed.Load(), err
}

// Run execute the ssh subcommand.
func (e *SSH) Run(clientAPI *api.API, cfg ClientConfig) error {
	for i := range e.SSHArgs {
		if cfg.SSH.SSHArgs[i] == "-F" {
			cfg.SSH.SSHConfFile = cfg.SSH.SSHArgs[i+1]
		}
	}
	if cfg.Socks.Target != "" {
		// we are in socks mode, so apply the socks option to the ssh
		cfg.SSH = SSH{
			Target:         cfg.Socks.Target,
			Socks:          cfg.Socks.Socks,
			HTTP:           cfg.SSH.HTTP,
			LocalSocksPort: cfg.Socks.LocalSocksPort,
			LocalHTTPPort:  cfg.Socks.LocalHTTPPort,
			SSH:            false,
			Print:          cfg.SSH.Print,
			Proxy:          false,
			SSHArgs:        cfg.Socks.SSHArgs,
			SSHConfFile:    cfg.ConfigFile,
		}
	}
	if e.Proxy {
		e.Socks = false
		e.HTTP = false
	}

	return e.Execute(clientAPI, cfg)
}

// Execute start the ssh.
func (e *SSH) Execute(clientAPI *api.API, cfg ClientConfig) error {
	if len(e.SSHArgs) == 1 && e.SSHArgs[0] == "" {
		e.SSHArgs = []string{}
	}
	agent, err := clientAPI.GetAgentByName(cfg.SSH.Target)
	if err != nil {
		log.Warn().Err(err).Str("target", cfg.SSH.Target).Msg("Failed to get agent")
		cfg.SSH.Target, err = GetFromSSHConfig(cfg.SSH.SSHConfFile, cfg.SSH.Target)
		if err != nil {
			log.Warn().Err(err).Str("SSH Config file", cfg.SSH.SSHConfFile).Str("target", cfg.SSH.Target).Msg("Failed to get agent")
		}

		log.Debug().Str("Target", cfg.SSH.Target).Msg("Trying using ssh_config file")

		agent, err = clientAPI.GetAgentByName(cfg.SSH.Target)
		if err != nil {
			return fmt.Errorf("failed to get agent by name (%s): %w", cfg.SSH.Target, err)
		}
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

	cmd := e.buildCommand(cfg, agent, exePath)
	if e.Print {
		//nolint:forbidigo
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
		}

		return err
	}

	return nil
}

// buildCommand build the ssh command.
func (e *SSH) buildCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	if e.Proxy {
		return e.buildTunnelSSHCommand(cfg, agent, exePath)
	}
	if e.SSH {
		return e.buildOuterSSHCommand(cfg, agent, exePath)
	}
	cmd := e.buildTunnelSSHCommand(cfg, agent, exePath)
	cmd.Args = append(cmd.Args, "-N")

	return cmd
}

// buildOuterSSHCommand build the outer SSH command. This SSH command will be executed in second
// through the ProxyCommand.
func (e *SSH) buildOuterSSHCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	innerCmd := e.buildTunnelSSHCommand(cfg, agent, exePath)
	cmd := Command{}
	cmd.Executable = "ssh"
	cmd.Args = buildAllSSHOptions(cfg)
	cmd.Env = buildEnvironments(cfg, "agent", exePath, cfg.SSH.Target)
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
	var proxyCmd string
	if e.Print {
		proxyCmd = fmt.Sprintf("-oProxyCommand=%s%s%s", sep, innerCmd.InlineEnv().String(), sep)
	} else {
		proxyCmd = fmt.Sprintf("-oProxyCommand=%s%s%s", sep, innerCmd.String(), sep)
	}
	cmd.Args = append(cmd.Args, proxyCmd)
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, agent.ID))
	cmd.Args = append(cmd.Args, cfg.SSH.SSHArgs...)

	return cmd
}

// buildTunnelSSHCommand create the ssh command used in the SSH proxycommand
// this SSH command is the tunnel one in the ssh command, but is actually the outer one
// when being executed (i.e.: it will be executed first).
func (e *SSH) buildTunnelSSHCommand(cfg ClientConfig, agent types.Agent, exePath string) Command {
	cmd := Command{
		Executable: "ssh",
	}
	cmd.Env = buildEnvironments(cfg, "otp", exePath, cfg.SSH.Target)
	for i := range cmd.Env {
		cmd.Env[i] = strings.ReplaceAll(cmd.Env[i], ` `, `\ `)
	}
	cmd.Args = buildInnerSSHOptions(cfg)
	cmd.Args = append(cmd.Args, "-p"+cfg.GetSshdPort())
	if e.SSH || e.Proxy {
		cmd.Args = append(cmd.Args, "-W127.0.0.1:"+agent.GetSSHPort())
	}
	if e.Socks && agent.GetSocksPort() != "0" && agent.GetSocksPort() != ":" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-L%d:127.0.0.1:%s", cfg.SSH.LocalSocksPort, agent.GetSocksPort()))
	}
	if e.HTTP && agent.GetHTTPPort() != "0" && agent.GetHTTPPort() != "/" {
		cmd.Args = append(cmd.Args, fmt.Sprintf("-L%d:127.0.0.1:%s", cfg.SSH.LocalHTTPPort, agent.GetHTTPPort()))
	}
	cmd.Args = append(cmd.Args, fmt.Sprintf("%s@%s", agent.Name, cfg.GetSshdHost()))

	return cmd
}

// buildEnvironments returns the environment variables required to access the agents.
func buildEnvironments(cfg ClientConfig, _ string, exePath string, target string) []string {
	envs := []string{
		"SSH_ASKPASS_REQUIRE=force",
		"SSH_ASKPASS=" + exePath,
		// prefixEnv("SERVER", cfg.Server),
		// prefixEnv("AGENT", target),
		// prefixEnv("TYPE", typePass),
		// prefixEnv("CONFIG_FILE", cfg.ConfigFile),
	}
	envs = append(envs, cfg.EnvVar(target)...)
	// if cfg.PrivatePassword != "" {
	// envs = append(envs, prefixEnv("PASSWORD", cfg.PrivatePassword))
	//}
	// if cfg.IsFlagInCommandLine("--access-token", "") {
	// envs = append(envs, prefixEnv("ACCESS_TOKEN", cfg.AccessToken))
	//}

	return envs
}

// buildAllSSHOptions returns the ssh options that are in common to the inner and the outer ssh.
func buildAllSSHOptions(cfg ClientConfig) []string {
	options := []string{
		"-oStrictHostKeyChecking=no",
		"-oUserKnownHostsFile=/dev/null",
		"-oPubkeyAuthentication=no",
		"-oPreferredAuthentications=password",
		"-oLogLevel=ERROR",
		"-oExitOnForwardFailure=no",
	}

	if cfg.Verbose > 0 {
		options = append(options, "-"+strings.Repeat("v", cfg.Verbose))
	}

	return options
}

// buildSshOptions returns the SSH options required to access the agents.
func buildInnerSSHOptions(cfg ClientConfig) []string {
	options := []string{
		"-oClearAllForwardings=no",
		// "-vv",
	}

	return append(options, buildAllSSHOptions(cfg)...)
}

// prefixEnv adds the application name to the provided value and returns it
// as an environment variable.
func prefixEnv(name string, value string) string {
	return fmt.Sprintf("%s_%s=%s", strings.ToUpper(common.AppName), strings.ToUpper(name), value)
}

// getPath returns the path of the binary currently being executed.
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
