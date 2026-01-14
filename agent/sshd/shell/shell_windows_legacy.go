//go:build windows && (amd64 || 386)

package shell

import (
	"io"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/ssh"
	"github.com/iamacarpet/go-winpty"
)

// runWithWinPTY start a pty using the legacy go-winpty method
func runWithWinPTY(s ssh.Session, cmd string, winCh <-chan ssh.Window) error {

	Env := append(os.Environ(),
		"PLATFORM="+strings.ToLower(runtime.GOOS),
		"USER="+s.User(),
		"LANG=en_US.UTF-8",
	)
	p, err := winpty.OpenWithOptions(winpty.Options{
		Command: cmd,
		Env:     Env,
	})
	if err != nil {
		return err
	}
	defer p.Close()

	// SSH → winpty
	go io.Copy(p.StdIn, s)

	// winpty → SSH
	go io.Copy(s, p.StdOut)

	// Resize handling
	go func() {
		for win := range winCh {
			p.SetSize(uint32(win.Width), uint32(win.Height))
		}
	}()

	<-s.Context().Done()
	return nil
}
