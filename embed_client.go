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
//go:embed agent client common server vendored/github.com/hazegard/socket.io-go.tar.gz vendored/github.com/aus/proxyplease@v0.1.0.tar.gz .goreleaser.yaml embed.go go.mod go.sum .env.build.tmpl scripts/garble.sh scripts/garble.bat scripts/shuffle/main.go
var Sources embed.FS

// GetRoot returns the directory path of the current source file.
func GetRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}

	return filepath.Dir(file)
}
