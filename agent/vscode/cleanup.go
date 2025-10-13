// Package vscode holds the agent side vscode functions
package vscode

import (
	"Goauld/common/utils"
	"Goauld/common/vscode"
	"errors"
	"os"
	"path/filepath"
)

// Cleanup tries to clean the vscode directory on the agent exit.
func Cleanup() error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.New("unable to get current working directory")
	}

	targetPath := filepath.Join(cwd, vscode.VSCode)

	if utils.IsDir(targetPath) {
		return os.RemoveAll(targetPath)
	}

	return nil
}
