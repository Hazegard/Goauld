package compiler

import (
	"io"
	"os"
	"strings"
)

// ReplaceInFile replace lines when a pattern is identified on the incoming content.
func ReplaceInFile(content string, pattern string, newContent string) string {
	lines := strings.Split(content, "\n")
	var newLines []string
	for _, line := range lines {
		if strings.HasPrefix(line, pattern) {
			newLines = append(newLines, newContent)
		} else {
			newLines = append(newLines, line)
		}
	}

	return strings.Join(newLines, "\n")
}

// CopyFile copies the file.
func CopyFile(src, dst string) error {
	// Open the source file for reading.
	srcStat, err := os.Stat(src)
	if err != nil {
		return err
	}
	//nolint:gosec
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer sourceFile.Close()

	// Create the destination file for writing.
	//nolint:gosec
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	//nolint:errcheck
	defer destinationFile.Close()

	// Copy the contents using io.Copy.
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}
	err = destinationFile.Chmod(srcStat.Mode())
	if err != nil {
		return err
	}

	// Ensure that any writes to the destination file are committed.
	return destinationFile.Sync()
}

// MkdirAll create a directory.
func MkdirAll(path string) error {
	return os.MkdirAll(path, 0o750)
}
