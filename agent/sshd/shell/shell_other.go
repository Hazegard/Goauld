//go:build !windows

package shell

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/ssh"
)

// ShellParam the param used to provide a shell command as a string.
const ShellParam = "-c"

// ShellLogin the param to start an interactive shell.
var ShellLogin = []string{"-l"}

// getShell return the first shell found on the system.
func getShell() Command {
	commands := []Command{
		{
			Executable: "bash",
		},
		{
			Executable: "zsh",
		},
		{
			Executable: "sh",
		},
	}
	cmd := getShellCmd(commands)
	if cmd.Executable != "" {
		return cmd
	}
	cmd = lookBinaries(commands)
	if cmd.Executable != "" {
		return cmd
	}

	return Command{
		Executable: "/bin/sh",
	}
}

// UpdateShell updates the shell command if required.
func UpdateShell(shell Command, rawCommand string) (Command, func() error, error) {
	if rawCommand != "" {
		shell.Args = []string{ShellParam, rawCommand}
	} else {
		shell.Args = ShellLogin
	}

	return shell, func() error { return nil }, nil
}

var defaultPaths = []string{
	"/bin",
	"/usr/bin",
	"/sbin",
	"/usr/sbin",
}

// lookBinaries returns the first Command found in the paths.
func lookBinaries(commands []Command) Command {
	for _, cmd := range commands {
		c := lookPaths(cmd, defaultPaths)
		if c.Executable != "" {
			return c
		}
	}

	return Command{}
}

// lookPaths returns whether the provided command if found in the provided paths.
func lookPaths(binaryName Command, paths []string) Command {
	for _, path := range paths {
		absCmd, err := lookPath(binaryName, path)
		if err != nil {
			continue
		}
		if absCmd.Executable != "" {
			return absCmd
		}
	}

	return Command{}
}

// lookPath checks if a binary exists in the given directory path and returns its absolute path.
func lookPath(binaryName Command, path string) (Command, error) {
	// Combine the given path and binary name to form the full file path
	fullPath := filepath.Join(path, binaryName.Executable)

	// Check if the file exists at the full path
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Command{}, fmt.Errorf("binary '%s' not found in path '%s'", binaryName, path)
		}

		return Command{}, fmt.Errorf("error checking file: %w", err)
	}

	// Check if the file is executable
	if info.Mode()&0o111 == 0 {
		return Command{}, fmt.Errorf("binary '%s' is not executable", binaryName)
	}

	// Return the absolute path
	absolutePath, err := filepath.Abs(fullPath)
	if err != nil {
		return Command{}, fmt.Errorf("error resolving absolute path: %w", err)
	}

	return Command{Executable: absolutePath}, nil
}

// This is an attempt to use builtin charmbracelet/ssh pty
// Without success (see agent/sshd/sshd.go)
/*func SetSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid:  true,
		Setctty: true,
	}
}
*/

func isLegacyWindows() bool {
	return false
}

func runWithWinPTY(_ ssh.Session, _ string, _ <-chan ssh.Window) error {
	return errors.New("windows not supported")
}
