package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/goccy/go-yaml"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"
)

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

// Hack required because goccy/go-yaml CustomMarshaller does not accept empty result (either nil, empty []byte("")), etc.
const EMPTUYRL = "THISISANEMPTYURL"

// Taken from https://github.com/alecthomas/kong-yaml

// YAML parse the file as YAML, but ignore the value found in the YAML file
// if the corresponding value is found in the environment variables.
// This is done to have environment variable precedence over
// the configuration files

// YAMLKeepEnvVar resolve the YAML configuration file.
// It does not override values found in the environment variables
func YAMLKeepEnvVar(r io.Reader) (kong.Resolver, error) {
	return YAML(r, true)
}

// YAMLOverwriteEnvVar resolve the YAML configuration file.
// Values found in the file are overwriting the value found in the environment variable
func YAMLOverwriteEnvVar(r io.Reader) (kong.Resolver, error) {
	return YAML(r, false)
}

// YAML returns a kong resolver to load YAML configuration file
// overwriteEnvVar should be set to true if we want the values in the YAML file to take precedence over
// the environment variable. If not, the value should be set to false
func YAML(r io.Reader, overwriteEnvVar bool) (kong.Resolver, error) {
	decoder := yaml.NewDecoder(r)
	config := map[string]interface{}{}
	err := decoder.Decode(&config)
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
	v := config[strings.Join(path, "-")]
	// Check if value is a uint64 and convert it to int
	if v, ok := v.(uint64); ok {
		return int(v) // Convert uint64 to int
	}
	return v
}

func CustomUrlP(u *url.URL) ([]byte, error) {

	if u.String() == "" {
		return []byte(EMPTUYRL), nil
	}
	return []byte((*u).String()), nil
}

// GenerateYAMLWithComments generates a YAML file with comments.
func GenerateYAMLWithComments(cfg any) (string, error) {

	comments := make(map[string][]*yaml.Comment)
	AddComments(cfg, "$", comments)
	yaml.RegisterCustomMarshaler(CustomUrlP)
	d, e := yaml.MarshalWithOptions(&cfg, yaml.WithComment(comments))
	//d, e := Marshal(cfg)
	if e != nil {
		return "", e
	}
	d = bytes.ReplaceAll(d, []byte("\n#"), []byte("\n\n# "))
	d = bytes.ReplaceAll(d, []byte(EMPTUYRL), []byte("\"\""))
	return string(d), e
}

// AddComments recursively traverses the struct, extracting comments based on the struct tags.
func AddComments(v interface{}, path string, comments map[string][]*yaml.Comment) {
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

	for i := 0; i < val.NumField(); i++ {
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
