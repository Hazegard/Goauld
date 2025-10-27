package ttyrec

import (
	"os/exec"
	"strings"
)

// getShellCmd return the first command found in the system path
func getShellCmd(cmds []string) string {
	for _, cmd := range cmds {
		if absPath, err := exec.LookPath(cmd); err == nil {
			return absPath
		}
	}
	return ""
}

func formatArgsWithShell(args []string, cmdParam string) []string {
	if len(args) == 0 {
		return args
	}
	return append([]string{cmdParam}, strings.Join(args, " "))
}
