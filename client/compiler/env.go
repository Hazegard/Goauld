package compiler

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ParseEnvFile reads an .env file and returns a slice of strings in "KEY=VALUE" format.
func ParseEnvFile(filepath string) ([]string, error) {
	envMap, err := parseAndResolveEnvFile(filepath)
	if err != nil {
		return nil, err
	}
	// Convert the map to slice of "key=value"
	var envs []string
	for k, v := range envMap {
		envs = append(envs, fmt.Sprintf("%s=%s", k, v))
	}

	return envs, nil
}

func parseAndResolveEnvFile(filePath string) (map[string]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	re := regexp.MustCompile(`^\s*([A-Za-z0-9_]+)\s*=\s*(.*)\s*$`)

	// First pass: Load all variables
	for scanner.Scan() {
		line := scanner.Text()

		// Skip comments and blank lines
		if strings.HasPrefix(line, "#") || strings.TrimSpace(line) == "" {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if len(matches) == 3 {
			key := matches[1]
			value := strings.Trim(matches[2], `"`)
			envMap[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Second pass: recursively resolve variable references
	resolved := make(map[string]string)
	for key := range envMap {
		resolved[key] = resolveValue(key, envMap, resolved, map[string]bool{})
	}

	return resolved, nil
}

var varRegex = regexp.MustCompile(`\$(\w+)`)

func resolveValue(key string, env map[string]string, resolved map[string]string, seen map[string]bool) string {
	// Prevent circular references
	if seen[key] {
		return ""
	}
	seen[key] = true

	val, exists := env[key]
	if !exists {
		// Fall back to actual environment
		return os.Getenv(key)
	}

	// Replace $VAR with its resolved value
	resolvedVal := varRegex.ReplaceAllStringFunc(val, func(match string) string {
		refKey := match[1:]
		// Try from already resolved map
		if v, ok := resolved[refKey]; ok {
			return v
		}
		// Try resolving now
		return resolveValue(refKey, env, resolved, seen)
	})

	resolved[key] = resolvedVal
	return resolvedVal
}
