//go:build windows

package exec

import (
	"os/exec"

	"golang.org/x/sys/windows"
)

func Backgrounize(cmd *exec.Cmd) *exec.Cmd {
	cmd.SysProcAttr = &windows.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP |
			windows.DETACHED_PROCESS |
			windows.CREATE_NO_WINDOW,
	}
	return cmd
}
