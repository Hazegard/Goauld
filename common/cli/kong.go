// Package cli holds the common cli
package cli

import (
	"Goauld/common/utils"
	"bytes"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/goccy/go-yaml"
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
	decoder := yaml.NewDecoder(r)
	config := map[string]any{}
	err := decoder.Decode(&config)
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("YAML agent decode error: %w", err)
	}

	return kong.ResolverFunc(func(_ *kong.Context, parent *kong.Path, flag *kong.Flag) (any, error) {
		if overwriteEnvVar || utils.Contains(ignoreOverwrite, flag.Name) {
			for _, env := range flag.Envs {
				_, ok := os.LookupEnv(env)
				if ok {
					//nolint:nilnil
					return nil, nil
				}
			}
		}
		// Build a string path up to this flag.
		var path []string
		for n := parent.Node(); n != nil && n.Type != kong.ApplicationNode; n = n.Parent {
			path = append([]string{n.Name}, path...)
		}
		path = append(path, flag.Name)

		return find(config, path), nil
	}), nil
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
	comments := make(map[string][]*yaml.Comment)
	AddComments(cfg, "$", comments)
	yaml.RegisterCustomMarshaler(CustomURLP)
	d, e := yaml.MarshalWithOptions(&cfg, yaml.WithComment(comments))
	if e != nil {
		return "", e
	}
	d = bytes.ReplaceAll(d, []byte("\n#"), []byte("\n\n# "))
	d = bytes.ReplaceAll(d, []byte(EMPTUYRL), []byte("\"\""))

	return string(d), e
}

// AddComments recursively traverses the struct, extracting comments based on the struct tags.
func AddComments(v any, path string, comments map[string][]*yaml.Comment) {
	val := reflect.ValueOf(v)
	typ := reflect.TypeOf(v)

	if val.Kind() == reflect.Pointer {
		val = val.Elem()
		typ = typ.Elem()
	}

	// If it's not a struct, return
	if val.Kind() != reflect.Struct {
		return
	}

	for i := range val.NumField() {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// Extract the YAML tag and the associated comment (help)
		tag := fieldType.Tag.Get("name")
		if tag != "" {
			// Look for "help" tag that contains the comment
			comment := fieldType.Tag.Get("help")
			splitComment := strings.Split(comment, "\n")
			if comment != "" {
				// Construct the path to the current field
				fullPath := path + "." + tag
				// Store the comment in the map with the full path
				comments[fullPath] = append(comments[fullPath], yaml.HeadComment(splitComment...))
			}
		}

		// Recursively inspect if the field is a struct or pointer to struct
		if field.Kind() == reflect.Struct || (field.Kind() == reflect.Pointer && field.Elem().Kind() == reflect.Struct) {
			AddComments(field.Interface(), path+"."+tag, comments)
		}
	}
}
