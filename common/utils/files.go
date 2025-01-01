package utils

import "os"

// GetCurrentDirectory returns the current directory from where the execution is started
func GetCurrentDirectory() (string, error) {
	exe, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return exe, nil
}
