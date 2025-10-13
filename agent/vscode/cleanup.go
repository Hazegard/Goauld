package vscode

import (
	"Goauld/common/utils"
	"Goauld/common/vscode"
	"fmt"
	"os"
	"path/filepath"
)

func Cleanup() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current working directory")
	}

	targetPath := filepath.Join(cwd, vscode.VSCode)

	if utils.IsDir(targetPath) {
		return os.RemoveAll(targetPath)
	}
	return nil
}
