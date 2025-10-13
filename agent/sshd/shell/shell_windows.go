//go:build windows

package shell

import (
	"Goauld/agent/sshd/shell/shimembed"
	"Goauld/common/log"
	"fmt"
)

const SHELL_PARAM = "/c"

var SHELL_LOGIN = []string{"-NoLogo", "-NoExit"} //[]string{"-l"}

// getShell return the first shell found on the system
func getShell() Command {
	commands := []Command{
		{
			Executable: "powershell",
			Args:       []string{"-Nologo"},
		},
		{
			Executable: "cmd",
		},
	}
	return getShellCmd(commands)
}

func UpdateShell(shell Command, rawCommand string) (Command, func() error, error) {
	if rawCommand != "" {
		shell.Args = []string{SHELL_PARAM, rawCommand}
		return shell, func() error { return nil }, nil
	} else {
		exe, cleanup, err := shimembed.DropShimSSHD()

		if err != nil {
			return Command{}, cleanup, fmt.Errorf("error while dropping sshd shim: %s", err)
		}
		log.Debug().Str("Path", exe).Msg("SSHD shim dropped")
		if shell.Executable == "powershell" {
			shell.Args = append(shell.Args, SHELL_LOGIN...)
		}
		if shell.Executable == "cmd" {
			shell.Args = append(shell.Args, "/K")
		}

		shell.Args = []string{shell.Executable}
		shell.Executable = exe
		return shell, cleanup, nil
	}
}

// This is an attempt to use builtin charmbracelet/ssh pty
// Without success (see agent/sshd/sshd.go)
/*func SetSysProcAttr(cmd *exec.Cmd) {
}
*/
