//go:build windows

package shell

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
