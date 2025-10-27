//go:build mini
// +build mini

// Package cli holds the common cli
package cli

import (
	"io"
	"net/url"
	"os"
	"strings"

	"github.com/alecthomas/kong"
)

// GetConfigFile returns the first existing directory.
func GetConfigFile(paths ...string) string {
	for _, path := range paths {
		_, err := os.Stat(kong.ExpandPath(path))
		if err != nil {
			continue
		}

		return path
	}

	return ""
}

// EMPTUYRL is a Hack required because goccy/go-yaml CustomMarshaller does not accept empty result (either nil, empty []byte("")), etc.
const EMPTUYRL = "THISISANEMPTYURL"

// Taken from https://github.com/alecthomas/kong-yaml

// YAML parse the file as YAML, but ignore the value found in the YAML file
// if the corresponding value is found in the environment variables.
// This is done to have environment variable precedence over
// the configuration files

// YAMLKeepEnvVar resolve the YAML configuration file.
// It does not override values found in the environment variables.
func YAMLKeepEnvVar(r io.Reader) (kong.Resolver, error) {
	return YAML(r, true, []string{})
}

// YAMLOverwriteEnvVar resolve the YAML configuration file.
// Values found in the file are overwriting the value found in the environment variable.
func YAMLOverwriteEnvVar(ignoreOverwrite []string) func(r io.Reader) (kong.Resolver, error) {
	return func(r io.Reader) (kong.Resolver, error) {
		return YAML(r, false, ignoreOverwrite)
	}
}

// YAML returns a kong resolver to load YAML configuration file
// overwriteEnvVar should be set to true if we want the values in the YAML file to take precedence over
// the environment variable. If not, the value should be set to false.
func YAML(r io.Reader, overwriteEnvVar bool, ignoreOverwrite []string) (kong.Resolver, error) {
	return nil, nil
}

func find(config map[string]any, path []string) any {
	if len(path) == 0 {
		return config
	}
	for i := range path {
		prefix := strings.Join(path[:i+1], "-")

		if child, ok := config[prefix].(map[string]any); ok {
			return find(child, path[i+1:])
		}
	}
	v := config[strings.Join(path, "-")]
	// Check if value is a uint64 and convert it to int

	if v, ok := v.(uint64); ok {
		//nolint:gosec
		return int(v) // Convert uint64 to int
	}

	return v
}

// CustomURLP returns an arbitrary string to represent an empty url when marshalling it in YAML.
func CustomURLP(u *url.URL) ([]byte, error) {
	if u.String() == "" {
		return []byte(EMPTUYRL), nil
	}

	return []byte((*u).String()), nil
}

// GenerateYAMLWithComments generates a YAML file with comments.
func GenerateYAMLWithComments(cfg any) (string, error) {
	return "", nil
}
