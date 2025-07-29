package shell

import (
	"Goauld/common/log"
	"context"
	"fmt"
	"github.com/aymanbagabas/go-pty"
	"github.com/charmbracelet/ssh"
	"io"
	"os"
	"os/exec"
	"strings"
)

// GivePty sets up a pseudo-terminal (PTY) for the given SSH session.
// It enables interaction with a shell (e.g., bash) through the session.
// If the session is not interactive, it directly executes the command, without
// wrapping it in a pty.
func GivePty(s ssh.Session, c []string, globalCtx context.Context) error {
	// Extract the PTY request and check if the session requested a PTY.
	if len(c) == 0 {
		cmd := getShell()
		c = cmd.Cli()
	}
	log.Debug().Msgf("Receving shell command [%s] (User: %s, RemoteAddr: %s)", strings.Join(c, " "), s.User(), s.RemoteAddr())

	// Get pty information
	ptyReq, winCh, isPty := s.Pty()
	if isPty {
		// This is an attempt to use builtin charmbracelet/ssh pty
		// Without success (see agent/sshd/sshd.go)
		/*
			cmd := exec.Command(c[0], c[1:]...)
			cmd.Env = append(os.Environ(), fmt.Sprintf("SSH_TTY=%s", ptyReq.Name()), fmt.Sprintf("TERM=%s", ptyReq.Term))

			if runtime.GOOS != "windows" {
				SetSysProcAttr(cmd)
			}
			err := ptyReq.Start(cmd)
			if err != nil {
				return err
			}
			if runtime.GOOS == "windows" {
				for cmd.ProcessState != nil {
					time.Sleep(100 * time.Millisecond)
				}
				s.Exit(cmd.ProcessState.ExitCode())
			} else {
				err := cmd.Wait()
				if err != nil {
					s.Exit(cmd.ProcessState.ExitCode())
				}
			}
			return nil
		*/
		// Get new pty
		pseudo, err := pty.New()
		if err != nil {
			return fmt.Errorf("error while opening pty: %s", err)
		}

		// Resize the pty to the client window
		w, h := ptyReq.Window.Width, ptyReq.Window.Height
		if err := pseudo.Resize(w, h); err != nil {
			return fmt.Errorf("error while resizing pty: %s", err)
		}

		// Exec the command within the pty
		cmd := pseudo.Command(c[0], c[1:]...)
		cmd.Env = append(os.Environ(), cmd.Env...)
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

		// Start a goroutine that waits for the end of the ssh session
		// And close the remaining readers and writers
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
	} else {
		// If no pty is requested, we execute the command directly
		cmd := exec.Command(c[0], c[1:]...)
		cmd.Stdout = s.Stderr()

		go func() {
			select {
			case <-globalCtx.Done():
			case <-s.Context().Done():
			}

			err := s.Close()
			if err != nil {
				log.Warn().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while closing session")
			}
		}()

		go func() {
			_, err := io.Copy(io.MultiWriter(cmd.Stdout, cmd.Stderr), s)
			if err != nil {
				log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while copying pty to client")
			}
		}()

		go func() {
			if cmd.Stdin != nil {
				_, err := io.Copy(s, cmd.Stdin)
				if err != nil {
					log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while copying input to pty")
				}
			}
		}()
		err := cmd.Start()
		if err != nil {
			log.Error().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while starting command")
		}

		// Wait until the session context is canceled.
		err = cmd.Wait()
		if err != nil {
			log.Warn().Err(err).Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("error while closing session")
		}
		log.Debug().Str("ID", s.User()).Str("Remote", s.RemoteAddr().String()).Msg("session closed")
	}
	return nil
}

type Command struct {
	Executable string
	Args       []string
}

func (c *Command) Cli() []string {
	return append([]string{c.Executable}, c.Args...)
}

// getShellCmd return the first command found in the system path
func getShellCmd(cmds []Command) Command {
	for _, cmd := range cmds {
		if absPath, err := exec.LookPath(cmd.Executable); err == nil {
			return Command{Executable: absPath, Args: cmd.Args}
		}
	}
	return Command{}
}
