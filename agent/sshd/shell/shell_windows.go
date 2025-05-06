//go:build windows

package shell

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
