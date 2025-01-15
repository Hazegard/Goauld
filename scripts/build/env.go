package main

import (
	"Goauld/common/crypto"
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"
)

func GenEnvFile(envFile string, cfg BuildConfig) (string, error) {
	newEnv := fmt.Sprintf("%s.%s", envFile, time.Now().Format("2006-01-02T15:04:05"))
	err := copyFile(envFile, newEnv)
	if err != nil {
		return "", err
	}

	bytes, err := os.ReadFile(newEnv)
	if err != nil {
		return "", err
	}
	content := string(bytes)

	if cfg.GenAgeKey {
		pubKey, privKey, err := genAgeKey()
		if err != nil {
			return "", err
		}
		content = replaceInFile(content, "SERVER__AGE_PRIVKEY", fmt.Sprintf("SERVER__AGE_PRIVKEY=%s", pubKey))
		content = replaceInFile(content, "AGENT__AGE_PUBKEY", fmt.Sprintf("AGENT__AGE_PUBKEY=%s", privKey))
	}

	if cfg.GenAccessToken {
		newtoken, err := crypto.GeneratePassword(42)
		if err != nil {
			return "", err
		}
		content = replaceInFile(content, "SERVER__ACCESS_TOKEN", fmt.Sprintf("SERVER__ACCESS_TOKEN=%s", newtoken))
	}

	return envFile, os.WriteFile(newEnv, []byte(content), 0700)
}

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
