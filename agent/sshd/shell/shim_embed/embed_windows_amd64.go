//go:build windows && amd64

package shim_embed

import _ "embed"

//go:embed sshd_shim/sshd_windows_amd64.exe
var EmbeddedBinary []byte
