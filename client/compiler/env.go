package compiler

import (
	"bufio"
	"os"
	"strings"
)

// ParseEnvFile reads an .env file and returns a slice of strings in "KEY=VALUE" format.
func ParseEnvFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var envs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Ignore comments and empty lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Ensure the line is in KEY=VALUE format
		if strings.Contains(line, "=") {
			envs = append(envs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envs, nil
}
