package Sources

import (
	"embed"
	"path/filepath"
	"runtime"
)

//go:embed agent client common server .goreleaser.yaml embed.go go.mod go.sum .env.build.tmpl scripts/garble.sh scripts/garble.bat
var Sources embed.FS

func GetRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Dir(file)
}
