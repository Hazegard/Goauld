package utils

import (
	"fmt"
	"os"
)

// GetCurrentDirectory returns the current directory from where the execution is started
func GetCurrentDirectory() (string, error) {
	exe, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return exe, nil
}

func OverwriteFile(path string, data []byte) error {
	// Get current file info (to preserve permissions)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	perm := info.Mode().Perm()

	// Open the file with write-only, truncate and no create flags
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open file for overwrite: %w", err)
	}
	defer file.Close()

	// Write new content
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}
