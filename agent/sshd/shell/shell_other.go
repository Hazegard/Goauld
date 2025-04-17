//go:build !windows

package shell

import (
	"fmt"
	"os"
	"path/filepath"
)

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

var defaultPaths = []string{
	"/bin",
	"/usr/bin",
	"/sbin",
	"/usr/sbin",
}

func lookBinaries(commands []Command) Command {
	for _, cmd := range commands {
		c := lookPaths(cmd, defaultPaths)
		if c.Executable != "" {
			return c
		}
	}
	return Command{}
}
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

// lookPath checks if a binary exists in the given directory path and returns its absolute path
func lookPath(binaryName Command, path string) (Command, error) {
	// Combine the given path and binary name to form the full file path
	fullPath := filepath.Join(path, binaryName.Executable)

	// Check if the file exists at the full path
	info, err := os.Stat(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return Command{}, fmt.Errorf("binary '%s' not found in path '%s'", binaryName, path)
		}
		return Command{}, fmt.Errorf("error checking file: %v", err)
	}

	// Check if the file is executable
	if info.Mode()&0111 == 0 {
		return Command{}, fmt.Errorf("binary '%s' is not executable", binaryName)
	}

	// Return the absolute path
	absolutePath, err := filepath.Abs(fullPath)
	if err != nil {
		return Command{}, fmt.Errorf("error resolving absolute path: %v", err)
	}

	return Command{Executable: absolutePath}, nil
}
