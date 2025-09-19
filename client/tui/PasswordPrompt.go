package tui

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"
)

// Prompt displays an SSH-like prompt to fetch from the command line the static agent password
func Prompt(agent string) (string, error) {
	fmt.Printf("(%s) Password: ", agent)
	bytePassword, err := term.ReadPassword(os.Stdin.Fd())
	if err != nil {
		return "", err
	}
	fmt.Println() // move to the next line after input
	return string(bytePassword), nil
}
