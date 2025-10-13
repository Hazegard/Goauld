//go:build windows

package main

import (
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	if len(os.Args) < 2 {
		return
	}

	exe := os.Args[1]

	args := []string{}
	if len(os.Args) > 2 {
		args = os.Args[2:]
	}

	cmd := exec.Command(exe, args...)
	// Forward stdio
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Make child in a new process group so it can receive CTRL events if needed
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("failed to start %s: %v", os.Args[0], err)
	}

	err := cmd.Wait()
	if err != nil {
		// try to map exit code
		//nolint:forcetypeassert
		if exitErr, ok := err.(*exec.ExitError); ok {
			if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(ws.ExitStatus())
			}
		}
		// unknown wait error
		log.Fatalf("powershell exited with error: %v", err)
	} else {
		if ps := cmd.ProcessState; ps != nil {
			//nolint:forcetypeassert
			if ws, ok := ps.Sys().(syscall.WaitStatus); ok {
				os.Exit(ws.ExitStatus())
			}
		}
	}

}
