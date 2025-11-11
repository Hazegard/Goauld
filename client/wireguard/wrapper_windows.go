//go:build windows

package wireguard

import (
	"path/filepath"
	"strings"
)

const WGCommand = "wireguard"

const DownFlag = "/uninstalltunnelservice"
const UpFlag = "/installtunnelservice"

func DownCmd(cfg string) []string {
	name := filepath.Base(cfg)
	name = strings.TrimSuffix(name, filepath.Ext(name))
	return []string{WGCommand, DownFlag, name}
}

func DownCmdString(cfg string) string {
	return strings.Join(DownCmd(cfg), " ")
}

func UpCmd(cfg string) []string {
	return []string{WGCommand, UpFlag, cfg}
}

func UpCmdString(cfg string) string {
	return strings.Join(UpCmd(cfg), " ")
}

func LatestHandshakes(file string) []string {
	return []string{"wg", "show", file, "latest-handshakes"}
}
