package shell

import (
	"fmt"
	"github.com/aymanbagabas/go-pty"
	"github.com/gliderlabs/ssh"
	"io"
	"log"
	"os/exec"
	"runtime"
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
	log.Println(c)
	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		pseudo, err := pty.New()
		if err != nil {
			return fmt.Errorf("error while opening pty: %s", err)
		}
		defer func(pseudo pty.Pty) {
			err := pseudo.Close()
			if err != nil {
				log.Printf("error while closing pty: %s", err)
			}
		}(pseudo)

		w, h := ptyReq.Window.Width, ptyReq.Window.Height

		if err := pseudo.Resize(w, h); err != nil {
			return fmt.Errorf("error while resizing pty: %s", err)
		}
		cmd := pseudo.Command(c[0], c[1:]...)
		log.Printf(ptyReq.Term)
		cmd.Env = append(cmd.Env, "TERM="+ptyReq.Term, "SSH_TTY="+pseudo.Name())
		if err := cmd.Start(); err != nil {
			return fmt.Errorf("error while starting command (%s): %s", c, err)
		}

		go func() {
			for win := range winCh {
				pseudo.Resize(win.Width, win.Height)
			}
		}()

		go func() {
			<-s.Context().Done() // Wait until the session context is canceled.
			log.Printf("Session ended: %s\n", s.RemoteAddr())
		}()

		go func() {
			_, err := io.Copy(pseudo, s)
			if err != nil {
				log.Printf("Error copying input to PTY: %v\n", err)
			}
		}()

		go func() {
			_, err := io.Copy(s, pseudo)
			if err != nil {
				log.Printf("Error copying input to PTY: %v\n", err)
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
