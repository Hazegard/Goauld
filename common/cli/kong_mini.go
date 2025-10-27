//go:build mini
// +build mini

// Package cli holds the common cli
package cli

import (
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

// ExpandPath is a helper function to expand a relative or home-relative path to an absolute path.
//
// eg. ~/.someconf -> /home/alec/.someconf
func ExpandPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if strings.HasPrefix(path, "~/") {
		u, err := user.Current()
		if err != nil {
			return path
		}
		return filepath.Join(u.HomeDir, path[2:])
	}
	abspath, err := filepath.Abs(path)
	if err != nil {
		return path
	}
	return abspath
}

// GetConfigFile returns the first existing directory.
func GetConfigFile(paths ...string) string {
	for _, path := range paths {
		_, err := os.Stat(ExpandPath(path))
		if err != nil {
			continue
		}

		return path
	}

	return ""
}
