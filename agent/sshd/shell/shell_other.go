//go:build !windows

package shell

func getShell() Command {
	commands := []Command{
		{
			Executable: "bash",
		},
		{
			Executable: "zsh",
		},
		{
			Executable: "sh",
		},
	}
	return getShellCmd(commands)
}
