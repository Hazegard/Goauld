package shell

import "os/exec"

func getShellCmd(cmds []string) string {
	for _, cmd := range cmds {
		if absPath, err := exec.LookPath(cmd); err == nil {
			return absPath
		}
	}
	return ""
}
