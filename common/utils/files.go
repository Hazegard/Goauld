package utils

import "os"

func GetCurrentDirectory() (string, error) {
	exe, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return exe, nil
}
