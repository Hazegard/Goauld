//go:build windows && !amd64 && !386

package shell

import (
	"errors"
	"runtime"

	"github.com/charmbracelet/ssh"
)

// runWithWinPTY shim on modern windows version
func runWithWinPTY(s ssh.Session, cmd string, winCh <-chan ssh.Window) error {

	return errors.New("Not implemented on windows " + runtime.GOARCH)
}
