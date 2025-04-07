//go:build !windows

package shell

func getShell() []string {
	return getShellCmd([]string{"bash", "zsh", "sh"})
}
