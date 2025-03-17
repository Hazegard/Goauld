package Sources

import "embed"

//go:embed agent client common server .goreleaser.yaml embed.go go.mod go.sum
var Sources embed.FS
