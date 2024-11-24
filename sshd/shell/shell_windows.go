package shell

import (
	"fmt"
	"github.com/gliderlabs/ssh"
	"io"
	"log"
)

func GivePty(s ssh.Session, cmd []string) {
	ptyReq, winCh, isPty := s.Pty()

	// Assume PTY is always allocated
	f, err := console.New(80, 24) // Use the default size, or optionally set dynamically based on ptyReq

	if err != nil {
		fmt.Fprint(s, "unable to create console", err)
		return
	}
	defer f.Close()

	// Decide whether you need these environment variables based on the actual situation
	if isPty {
		f.SetENV([]string{"TERM=" + ptyReq.Term})
	}

	args := []string{"cmd.exe"}
	if len(s.Command()) > 0 {
		args = append(args, "/c")
		args = append(args, s.Command()...)
	}
	err = f.Start(args)
	if err != nil {
		fmt.Fprint(s, "unable to start", args, err)
		return
	}
	log.Println(args)

	done := s.Context().Done()
	go func() {
		<-done
		log.Println(s.RemoteAddr(), "done")
		f.Close()
	}()

	// Monitor window size changes; remove this part if unnecessary
	if isPty {
		go func() {
			for win := range winCh {
				f.SetSize(win.Width, win.Height)
			}
		}()
	}

	// Handle input/output
	go func() {
		io.Copy(f, s) // stdin
	}()
	io.Copy(s, f) // stdout

	if _, err := f.Wait(); err != nil {
		log.Println(args[0], err)
	}
}
