//go:build !client && !mini

// Package sources holds the agent source code
package sources

import (
	"embed"
	"path/filepath"
	"runtime"
)

var Sources embed.FS

// GetRoot returns the directory path of the current source file.
func GetRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	return filepath.Dir(file)
}
