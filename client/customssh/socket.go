package customssh

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"time"
)

// CheckControlSocket checks whether an SSH ControlMaster socket is valid.
// userHost must match the original master invocation (e.g. "user@host").
//
// Returns:
//
//	true, nil   -> socket is valid
//	false, nil  -> socket is stale / invalid
//	false, err  -> ssh execution error (not protocol failure)
func CheckControlSocket(socketPath, userHost string) (bool, error) {
	// Fast fail: path must exist
	if _, err := os.Stat(socketPath); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}

		return false, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(
		ctx,
		"ssh",
		"-S", socketPath,
		"-O", "check",
		userHost,
	)

	// Silence output
	cmd.Stdout = &bytes.Buffer{}
	cmd.Stderr = &bytes.Buffer{}

	err := cmd.Run()

	// Exit code 0 → valid socket
	if err == nil {
		return true, nil
	}

	// ssh returns exit status 255 for invalid/stale sockets
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 0 {
			return false, nil
		}
	}

	return false, err
}
