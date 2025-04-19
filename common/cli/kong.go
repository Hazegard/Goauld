package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"strings"
)

// Taken from https://github.com/alecthomas/kong-yaml

// YAML parse the file as YAML, but ignore the value found in the YAML file
// if the corresponding value is found in the environment variables.
// This is done to have environment variable precedence over
// the configuration files

func YAMLKeepEnvVar(r io.Reader) (kong.Resolver, error) {
	return YAML(r, true)
}

func YAMLOverwriteEnvVar(r io.Reader) (kong.Resolver, error) {
	return YAML(r, false)
}

func YAML(r io.Reader, overwriteEnvVar bool) (kong.Resolver, error) {
	decoder := yaml.NewDecoder(r)
	config := map[string]interface{}{}
	err := decoder.Decode(config)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("YAML agent decode error: %w", err)
	}
	return kong.ResolverFunc(func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		if overwriteEnvVar {
			for _, env := range flag.Envs {
				_, ok := os.LookupEnv(env)
				if ok {
					return nil, nil
				}
			}
		}
		// Build a string path up to this flag.
		path := []string{}
		for n := parent.Node(); n != nil && n.Type != kong.ApplicationNode; n = n.Parent {
			path = append([]string{n.Name}, path...)
		}
		path = append(path, flag.Name)
		//path = strings.Split(strings.Join(path, "-"), "-")
		return find(config, path), nil
	}), nil
}

func find(config map[string]interface{}, path []string) interface{} {
	if len(path) == 0 {
		return config
	}
	for i := 0; i < len(path); i++ {
		prefix := strings.Join(path[:i+1], "-")
		if child, ok := config[prefix].(map[string]interface{}); ok {
			return find(child, path[i+1:])
		}
	}
	return config[strings.Join(path, "-")]
}

// GenerateYAMLWithComments generates a YAML file with comments.
func GenerateYAMLWithComments(cfg any) (string, error) {
	d, e := Marshal(cfg)
	if e != nil {
		return "", e
	}
	d = bytes.ReplaceAll(d, []byte("\n#"), []byte("\n\n#"))
	return string(d), e
}
