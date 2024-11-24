package shell

import (
	"fmt"
	"github.com/aymanbagabas/go-pty"
	"github.com/gliderlabs/ssh"
	"io"
	"log"
)

// GivePty sets up a pseudo-terminal (PTY) for the given SSH session.
// It enables interaction with a shell (e.g., bash) through the session.
func GivePty(s ssh.Session, c []string) error {
	// Extract PTY request and check if the session requested a PTY.
	if len(c) == 0 {
		c = []string{getShellCmd([]string{"bash", "zsh", "sh"})}
	}
	log.Println(c)
	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		pseudo, err := pty.New()
		if err != nil {
			return fmt.Errorf("error while opening pty: %s", err)
		}
		defer pseudo.Close()
		w, h := ptyReq.Window.Width, ptyReq.Window.Height

		if err := pseudo.Resize(w, h); err != nil {
			return fmt.Errorf("error while resizing pty: %s", err)
		}
		cmd := pseudo.Command(c[0], c[1:]...)
		log.Printf(ptyReq.Term)
		cmd.Env = append(cmd.Env) //, "TERM="+ptyReq.Term) // "SSH_TTY="+pseudo.Name(),
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
	/*
		// Start a bash shell in a new PTY.
		shell := exec.Command(cmd[0], cmd[1:]...)
		ptmx, err := pty.Start(shell) // Start the PTY with the shell process.
		if err != nil {
			log.Printf("Failed to start PTY: %v\n", err)
			_, _ = io.WriteString(s, "Failed to start shell.\n")
			return
		}
		// Ensure PTY and shell resources are released on function exit.
		defer func() {
			_ = ptmx.Close() // Close the PTY.
			_ = shell.Wait() // Wait for the shell to exit.
		}()

		// Configure the initial PTY window size if the request includes it.
		if isPty {
			err = pty.Setsize(ptmx, &pty.Winsize{
				Rows: uint16(ptyReq.Window.Height),
				Cols: uint16(ptyReq.Window.Width),
			})
			if err != nil {
				log.Printf("Failed to set initial window size: %v\n", err)
			} else {
				log.Printf("PTY initialized with TERM=%s, size=%dx%d\n",
					ptyReq.Term, ptyReq.Window.Width, ptyReq.Window.Height)
			}
		}

		// Log the shell start.
		log.Printf("Shell started: %s\n", shell.Path)

		// Handle session context cancellation (e.g., client disconnects).
		go func() {
			<-s.Context().Done() // Wait until the session context is canceled.
			log.Printf("Session ended: %s\n", s.RemoteAddr())
		}()

		// Monitor and apply window size changes sent by the client.
		go func() {
			for win := range winCh {
				err := pty.Setsize(ptmx, &pty.Winsize{
					Rows: uint16(win.Height),
					Cols: uint16(win.Width),
				})
				if err != nil {
					log.Printf("Failed to resize PTY: %v\n", err)
				}
			}
		}()

		// Redirect input from the session to the PTY (stdin).
		go func() {
			_, err := io.Copy(ptmx, s)
			if err != nil {
				log.Printf("Error copying input to PTY: %v\n", err)
			}
		}()

		// Redirect output from the PTY to the session (stdout/stderr).
		_, err = io.Copy(s, ptmx)
		if err != nil {
			log.Printf("Error copying output from PTY: %v\n", err)
		}

	*/
}
