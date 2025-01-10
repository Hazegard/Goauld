package shell

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"strings"

	"Goauld/common/log"

	"github.com/aymanbagabas/go-pty"
	"github.com/gliderlabs/ssh"
)

// GivePty sets up a pseudo-terminal (PTY) for the given SSH session.
// It enables interaction with a shell (e.g., bash) through the session.
func GivePty(s ssh.Session, c []string, globalCtx context.Context) error {
	// Extract PTY request and check if the session requested a PTY.
	if len(c) == 0 {
		if runtime.GOOS == "windows" {
			c = getShellCmd([]string{"powershell", "cmd"})
		} else {
			c = getShellCmd([]string{"bash", "zsh", "sh"})
		}
	}
	log.Debug().Msgf("Receving shell command [%s] (User: %s, RemoteAddr: %s)", strings.Join(c, " "), s.User(), s.RemoteAddr())

	// Get pty information
	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		// Get new pty
		pseudo, err := pty.New()
		if err != nil {
			return fmt.Errorf("error while opening pty: %s", err)
		}
		defer func() {
			err := pseudo.Close()
			if err != nil {
				log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while closing pty")
			}
		}()

		// Resize the pty to the client window
		w, h := ptyReq.Window.Width, ptyReq.Window.Height
		if err := pseudo.Resize(w, h); err != nil {
			return fmt.Errorf("error while resizing pty: %s", err)
		}

		// Exec the command within the pty
		cmd := pseudo.Command(c[0], c[1:]...)
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term, "SSH_TTY="+pseudo.Name())
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error while starting command (%s): %s", c, err)
		}
		// start a loop to handle dynamic window size modification
		go func() {
			for win := range winCh {
				err := pseudo.Resize(win.Width, win.Height)
				if err != nil {
					log.Trace().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while resizing pty")
				}
			}
		}()

		go func() {
			select {
			case <-globalCtx.Done():
			case <-s.Context().Done():
			}

			err = s.Close()
			if err != nil {
				log.Warn().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while closing session")
			}
			// Wait until the session context is canceled.
			err = pseudo.Close()
			if err != nil {
				log.Warn().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while closing pty")
			}
			log.Debug().Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("session closed")
		}()

		go func() {
			_, err := io.Copy(pseudo, s)
			if err != nil {
				log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while copying pty to client")
			}
		}()

		go func() {
			_, err := io.Copy(s, pseudo)
			if err != nil {
				log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while copying input to pty")
			}
		}()

		if err := cmd.Wait(); err != nil {
			// return fmt.Errorf("error while waiting for command (%s): %s", c, err)
		}
	}
	return nil
}

// getShellCmd return the first command found in the system path
func getShellCmd(cmds []string) []string {
	for _, cmd := range cmds {
		if absPath, err := exec.LookPath(cmd); err == nil {
			return []string{absPath}
		}
	}
	return []string{}
}
