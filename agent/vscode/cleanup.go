package vscode

import (
	"Goauld/common/utils"
	"fmt"
	"os"
	"path/filepath"
)

func Cleanup() error {
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("unable to get current working directory")
	}

	targetPath := filepath.Join(cwd, "tealc-VSServer")

	if utils.IsDir(targetPath) {
		return os.RemoveAll(targetPath)
	}
	return nil
}
