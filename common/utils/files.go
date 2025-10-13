//nolint:revive
package utils

import (
	"Goauld/common/log"
	"fmt"
	"os"
	"runtime"
)

func closeFile(file *os.File) {
	_ = file.Close()
}

// WriteToFile writes data the path.
func WriteToFile(content string, outputFile string) error {
	//nolint:gosec
	f, err := os.Create(outputFile)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", outputFile, err)
	}
	_, err = f.WriteString(content)
	if err != nil {
		return fmt.Errorf("error writing to file %s: %w", outputFile, err)
	}
	defer closeFile(f)

	return nil
}

// OverwriteFile writes data the path and overwrite existing data if needed.
func OverwriteFile(path string, data []byte) error {
	// Get current file info (to preserve permissions)
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	perm := info.Mode().Perm()

	// Open the file with write-only, truncate and no create flags
	//nolint:gosec
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to open file for overwrite: %w", err)
	}
	//nolint:errcheck
	defer file.Close()

	// Write new content
	_, err = file.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

// CreateOrReplaceFileSymlink creates a symbolic link to a file.
// If the destination exists, it is removed first.
// On Windows, if symlink creation fails, falls back to a hard link.
func CreateOrReplaceFileSymlink(target, linkName string) error {
	// Remove existing file/link if it exists
	if _, err := os.Lstat(linkName); err == nil {
		if removeErr := os.Remove(linkName); removeErr != nil {
			return fmt.Errorf("failed to remove existing file/link: %w", removeErr)
		}
	}

	// Try to create the symlink
	err := os.Symlink(target, linkName)
	if err == nil {
		return nil
	}

	// On Windows, fall back to hard link if symlink fails
	if runtime.GOOS == "windows" {
		if linkErr := os.Link(target, linkName); linkErr == nil {
			log.Warn().Str("target", target).Str("linkname", linkName).Msg("Symlink not allowed; created a hard link instead")

			return nil
		}
	}

	return fmt.Errorf("failed to create symlink: %w", err)
}

// IsDir takes a path as input and returns true if the path exists and is a directory, false otherwise.
func IsDir(path string) bool {
	// Get the file info for the given path
	info, err := os.Stat(path)
	if err != nil {
		// If there's an error (e.g., the file doesn't exist), return false
		return false
	}

	// Return true if it's a directory, false otherwise
	return info.IsDir()
}
