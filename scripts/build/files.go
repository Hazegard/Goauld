package main

import (
	"io"
	"os"
	"strings"
)

func replaceInFile(content string, pattern string, new string) string {
	lines := strings.Split(content, "\n")
	newLines := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, pattern) {
			newLines = append(newLines, new)
		} else {
			newLines = append(newLines, line)
		}
	}
	return strings.Join(newLines, "\n")
}

func copyFile(src, dst string) error {
	// Open the source file for reading.
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file for writing.
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents using io.Copy.
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Ensure that any writes to the destination file are committed.
	return destinationFile.Sync()
}

func MkdirAll(path string) error {
	return os.MkdirAll(path, 0o755)
}
