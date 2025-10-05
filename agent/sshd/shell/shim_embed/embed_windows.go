package shim_embed

import (
	"Goauld/common"
	"fmt"
	"os"
	"path/filepath"
)

// DropShimSSHD writes an embedded sshd.exe binary to the current directory if possible.
// If that fails (e.g., due to permissions), it falls back to a temp dir.
// Returns the full path to the written binary, the directory used, and an error if any.
func DropShimSSHD() (string, func() error, error) {
	const fileName = "sshd.exe"

	cleanNone := func() error {
		return nil
	}

	// 1) Try current working directory first
	cwd, err := os.Getwd()
	if err == nil {
		targetBinary := filepath.Join(cwd, fileName)
		if err := os.WriteFile(targetBinary, EmbeddedBinary, 0o755); err == nil {

			cleanup := func() error {
				return os.Remove(targetBinary)
			}
			return targetBinary, cleanup, nil
		}
	}

	// 2) Fallback to a temp dir
	tmpDir, err := os.MkdirTemp("", fmt.Sprintf("%s-sshd-*", common.AppName()))
	if err != nil {
		return "", cleanNone, fmt.Errorf("mkdirtemp: %w", err)
	}

	cleanDir := func() error {
		return os.RemoveAll(tmpDir)
	}

	targetPath := filepath.Join(tmpDir, fileName)
	if err := os.WriteFile(targetPath, EmbeddedBinary, 0o755); err != nil {
		return "", cleanDir, fmt.Errorf("write sshd.exe: %w", err)
	}

	return targetPath, cleanDir, nil
}
