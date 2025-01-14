package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"
)

// Taken from https://github.com/alecthomas/kong-yaml

// YAML parse the file as yaml, but ignore the value found in the YAML file
// if the corresponding value is found in the environment variables.
// This is done in order to have environment variable precedence over
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
		path = strings.Split(strings.Join(path, "-"), "-")
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

// generateYAMLWithComments generates a YAML file with comments.
func GenerateYAMLWithComments(cfg any) (string, error) {
	var node yaml.Node

	// Marshal the struct into a yaml.Node.
	data, err := MarshalYAML(cfg)
	if err != nil {
		return "", fmt.Errorf("failed to marshal struct: %w", err)
	}
	if err := yaml.Unmarshal(data, &node); err != nil {
		return "", fmt.Errorf("failed to unmarshal back into yaml.Node: %w", err)
	}

	nameHelpMap := make(map[string]string)
	//v := reflect.ValueOf(cfg)
	t := reflect.TypeOf(cfg)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		//value := v.Field(i)
		//fmt.Println(field)
		//fmt.Println(value)
		// Get the "name" tag.
		help := field.Tag.Get("help")
		if help == "" {
			continue // Skip fields without the "name" tag.
		}
		name := field.Tag.Get("name")
		if name == "" {
			continue // Skip fields without the "name" tag.
		}
		nameHelpMap[name] = help
	}
	//node.Content[0].Content[i].HeadComment = tag

	for i, n := range node.Content[0].Content {
		v := n.Value
		n.HeadComment = nameHelpMap[v]
		if i != 0 && n.HeadComment != "" {
			n.HeadComment = fmt.Sprintf("\n%s", nameHelpMap[v])
		}
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(&node); err != nil {
		return "", fmt.Errorf("failed to encode yaml.Node: %w", err)
	}

	return buf.String(), nil
}

// MarshalYAML implements the yaml.Marshaler interface and uses reflection.
func MarshalYAML(c any) ([]byte, error) {
	v := reflect.ValueOf(c)
	t := reflect.TypeOf(c)

	// Create a map to hold the resulting YAML key-value pairs.
	mapped := make(map[string]interface{})

	// Iterate through the struct fields.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get the "name" tag.
		tag := field.Tag.Get("name")
		if tag == "" {
			continue // Skip fields without the "name" tag.
		}

		// Add the field value to the map using the tag as the key.
		mapped[tag] = value.Interface()
	}

	return yaml.Marshal(mapped)
}
