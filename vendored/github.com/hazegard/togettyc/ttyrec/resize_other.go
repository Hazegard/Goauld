//go:build !windows

package ttyrec

import (
	"github.com/runletapp/go-console"
	"golang.org/x/term"
	"log"
	"os"
	"os/signal"
	"syscall"
)

var shells = []string{"zsh", "bash", "sh"}

func handleResize(c console.Console) {
	// Handle pty size.
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGWINCH)
	for range ch {
		w, h, err := term.GetSize(int(os.Stdin.Fd()))
		if err != nil {
			log.Println(err)
			continue
		}
		err = c.SetSize(w, h)
		if err != nil {
			log.Println(err)
		}

	}
	ch <- syscall.SIGWINCH                        // Initial resize.
	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signals when done.

}

func detectShell() string {
	shell := os.Getenv("SHELL")
	return shell
}

func formatArgs(args []string) []string {
	return formatArgsWithShell(args, "-c")
}
