package cmd

import "os/exec"

// CheckCommands checks whether the provided commands are found in the path
// It returns all commands that are not found in the path.
func CheckCommands(cmds []string) []string {
	var notFound []string
	for _, cmd := range cmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			notFound = append(notFound, cmd)
		}
	}

	return notFound
}
