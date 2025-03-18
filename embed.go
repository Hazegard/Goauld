package Sources

import "embed"

//go:embed agent client common server .goreleaser.yaml embed.go go.mod go.sum .env.build.tmpl
var Sources embed.FS
