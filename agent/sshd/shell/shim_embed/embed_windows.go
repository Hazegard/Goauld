package shim_embed

import (
	"fmt"
	"os"
	"path/filepath"
)

// runShimForSession writes an embedded sshd.exe to a temp dir, executes it with its stdio
// connected to the provided conn (SSH channel), waits for it to exit, then cleans up.
func DropShimSSHD() (string, error) {
	// 1) create a temp dir (unique per session)
	tmpDir, err := os.MkdirTemp("", "myapp-sshd-*")
	if err != nil {
		return "", fmt.Errorf("mkdirtemp: %w", err)
	}
	// Ensure cleanup on function exit (we'll remove after process ends)

	// 2) write sshd.exe into that dir
	targetPath := filepath.Join(tmpDir, "sshd.exe")
	if err := os.WriteFile(targetPath, EmbeddedBinary, 0o755); err != nil {
		return "", fmt.Errorf("write sshd.exe: %w", err)
	}

	return targetPath, nil
}
