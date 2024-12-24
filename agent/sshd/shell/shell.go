package shell

import (
	"Goauld/common/log"
	"fmt"
	"github.com/aymanbagabas/go-pty"
	"github.com/gliderlabs/ssh"
	"io"
	"os/exec"
	"runtime"
	"strings"
)

// GivePty sets up a pseudo-terminal (PTY) for the given SSH session.
// It enables interaction with a shell (e.g., bash) through the session.
func GivePty(s ssh.Session, c []string) error {
	// Extract PTY request and check if the session requested a PTY.
	if len(c) == 0 {
		if runtime.GOOS == "windows" {
			c = getShellCmd([]string{"powershell", "cmd"})
		} else {
			c = getShellCmd([]string{"bash", "zsh", "sh"})
		}
	}
	log.Debug().Msgf("Receving shell command [%s] (User: %s, RemoteAddr: %s)", strings.Join(c, " "), s.User(), s.RemoteAddr())
	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		pseudo, err := pty.New()
		if err != nil {
			return fmt.Errorf("error while opening pty: %s", err)
		}
		defer func(pseudo pty.Pty) {
			err := pseudo.Close()
			if err != nil {
				log.Error().Err(err).Msg("error while closing pty")
			}
		}(pseudo)

		w, h := ptyReq.Window.Width, ptyReq.Window.Height

		if err := pseudo.Resize(w, h); err != nil {
			return fmt.Errorf("error while resizing pty: %s", err)
		}
		cmd := pseudo.Command(c[0], c[1:]...)
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term, "SSH_TTY="+pseudo.Name())
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error while starting command (%s): %s", c, err)
		}

		go func() {
			for win := range winCh {
				err := pseudo.Resize(win.Width, win.Height)
				if err != nil {
					log.Trace().Err(err).Msg("error while resizing pty")
				}
			}
		}()

		go func() {
			<-s.Context().Done() // Wait until the session context is canceled.
			log.Debug().Msgf("session closed (%s, %s)", s.User(), s.RemoteAddr())
		}()

		go func() {
			_, err := io.Copy(pseudo, s)
			if err != nil {
				log.Error().Err(err).Msgf("error while copying pty to client (%s, %s)", s.User(), s.RemoteAddr())
			}
		}()

		go func() {
			_, err := io.Copy(s, pseudo)
			if err != nil {
				log.Error().Err(err).Msgf("error while copying input to pty (%s, %s)", s.User(), s.RemoteAddr())
			}
		}()

		if err := cmd.Wait(); err != nil {
			return fmt.Errorf("error while waiting for command (%s): %s", c, err)
		}
	}
	return nil
}

func getShellCmd(cmds []string) []string {
	for _, cmd := range cmds {
		if absPath, err := exec.LookPath(cmd); err == nil {
			return []string{absPath}
		}
	}
	return []string{}
}
