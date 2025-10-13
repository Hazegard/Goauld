//go:build windows && arm

package shimembed

import _ "embed"

//go:embed sshd_shim/sshd_windows_arm.exe
var EmbeddedBinary []byte
