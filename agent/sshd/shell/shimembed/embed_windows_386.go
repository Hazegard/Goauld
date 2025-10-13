//go:build windows && 386

package shimembed

import _ "embed"

//go:embed sshd_shim/sshd_windows_386.exe
var EmbeddedBinary []byte
