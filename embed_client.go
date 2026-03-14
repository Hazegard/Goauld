//go:build client

// Package sources holds the agent source code
package sources

import (
	"embed"
	"path/filepath"
	"runtime"
)

// Sources contains the agent source code
// Sources embed the agent source code te be able to dynamically compile it
//
//nolint:revive
//go:embed agent client common server .goreleaser.yaml embed.go embed_mini.go embed_client.go go.mod go.sum .env.build.tmpl scripts/garble.sh scripts/garble.bat scripts/shuffle/main.go
var Sources embed.FS

// GetRoot returns the directory path of the current source file.
func GetRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	return filepath.Dir(file)
}
