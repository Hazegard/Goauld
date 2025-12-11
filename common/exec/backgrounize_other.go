//go:build linux || darwin

package exec

import (
	"os/exec"
	"syscall"
)

func Backgrounize(c *exec.Cmd) *exec.Cmd {
	c.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	return c
}
