package tui

import (
	"fmt"
	"github.com/charmbracelet/x/term"
	"os"
)

func Prompt(agent string) (string, error) {
	fmt.Printf("(%s) Password: ", agent)
	bytePassword, err := term.ReadPassword(os.Stdin.Fd())
	if err != nil {
		return "", err
	}
	fmt.Println() // move to next line after input
	return string(bytePassword), nil
}
