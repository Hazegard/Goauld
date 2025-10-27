//go:build windows

package ttyrec

import (
	"github.com/runletapp/go-console"
	"golang.org/x/sys/windows"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

var shells = []string{"powershell", "cmd"}

func getConsoleSize(handle windows.Handle) (width, height int, err error) {
	var csbi windows.ConsoleScreenBufferInfo
	err = windows.GetConsoleScreenBufferInfo(handle, &csbi)
	if err != nil {
		return 0, 0, err
	}
	width = int(csbi.Window.Right - csbi.Window.Left + 1)
	height = int(csbi.Window.Bottom - csbi.Window.Top + 1)
	return width, height, nil
}

func handleResize(c console.Console) {
	// Get console handle
	handle := windows.Handle(os.Stdout.Fd())

	// Create a signal channel for handling resize events
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	// Initial resize
	ch <- os.Interrupt

	go func() {
		for range ch {
			w, h, err := getConsoleSize(handle)
			if err != nil {
				log.Println("Error getting console size:", err)
				continue
			}
			err = c.SetSize(w, h)
			if err != nil {
				log.Println("Error setting console size:", err)
			}
		}
	}()

	defer func() { signal.Stop(ch); close(ch) }() // Cleanup signal handler
}

func detectShell() string {
	// Check for Unix-like systems (Linux/macOS)
	shell := os.Getenv("SHELL")
	if shell != "" {
		return filepath.Base(shell) // Extract only the shell name
	}

	// Check for Windows environment
	comSpec := os.Getenv("ComSpec")
	if strings.Contains(comSpec, "cmd.exe") {
		return "cmd"
	}

	// Check if PowerShell is being used
	if os.Getenv("PSModulePath") != "" {
		return "powershell"
	}

	// Check if running Git Bash
	if os.Getenv("MSYSTEM") != "" {
		return "git-bash"
	}

	// Default fallback
	return "unknown"
}

func formatArgs(args []string) []string {
	return formatArgsWithShell(args, "/c")
}
