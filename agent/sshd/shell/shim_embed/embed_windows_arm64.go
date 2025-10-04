//go:build windows && arm64

package shim_embed

import _ "embed"

//go:embed sshd_shim/sshd_windows_arm64.exe
var EmbeddedBinary []byte
