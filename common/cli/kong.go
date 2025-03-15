package cli

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"gopkg.in/yaml.v3"
	"io"
	"net/url"
	"os"
	"reflect"
	"strings"
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

func GenerateNameHelpMap(cfg any) map[string]string {

	nameHelpMap := make(map[string]string)
	// v := reflect.ValueOf(cfg)
	t := reflect.TypeOf(cfg)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// value := v.Field(i)
		// fmt.Println(field)
		// fmt.Println(value)
		// Get the "name" tag.
		help := field.Tag.Get("help")
		if help == "" {
			continue // Skip fields without the "name" tag.
		}
		name := field.Tag.Get("name")
		if name == "" {
			continue // Skip fields without the "name" tag.
		}
		if field.Type.Kind() == reflect.Struct {
			help = "" //GenerateNameHelpMap(field)
		}
		nameHelpMap[name] = help
	}
	return nameHelpMap
}

// generateYAMLWithComments generates a YAML file with comments.
func GenerateYAMLWithComments(cfg any) (string, error) {
	d, e := Marshal(cfg)
	if e != nil {
		return "", e
	}
	d = bytes.ReplaceAll(d, []byte("\n#"), []byte("\n\n#"))
	return string(d), e
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
	// v := reflect.ValueOf(cfg)
	t := reflect.TypeOf(cfg)

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		// value := v.Field(i)
		// fmt.Println(field)
		// fmt.Println(value)
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
	// node.Content[0].Content[i].HeadComment = tag

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

// MarshalYAML implements the yaml.Marshaller interface and uses reflection.
func MarshalYAML(c any) ([]byte, error) {
	mapped, err := marshalStruct(reflect.ValueOf(c))
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(mapped)
}

// marshalStruct recursively converts a struct (or pointer to struct) to a map[string]interface{} using the "name" tag.
func marshalStruct(v reflect.Value) (map[string]interface{}, error) {
	// Dereference pointers to get to the underlying struct.

	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, nil
		}
		v = v.Elem()
	}

	t := v.Type()
	mapped := make(map[string]interface{})
	// Precompute the pointer type for *url.URL.
	urlPtrType := reflect.TypeOf((*url.URL)(nil))

	// Iterate through the struct fields.
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		value := v.Field(i)

		// Get the "name" tag.
		tag := field.Tag.Get("name")
		if tag == "" {
			continue // Skip fields without the "name" tag.
		}

		// Handle *url.URL specially.
		if field.Type == urlPtrType {
			if value.IsNil() {
				mapped[tag] = "" // or nil, based on your preference
			} else {
				u := value.Interface().(*url.URL)
				mapped[tag] = u.String()
			}
			continue
		}
		// If the field is a nested struct, recursively marshal it.
		switch value.Kind() {
		case reflect.Struct:
			nested, err := marshalStruct(value)
			if err != nil {
				return nil, err
			}
			mapped[tag] = nested
		case reflect.Ptr:
			// If it's a pointer to a struct, and non-nil, process the underlying struct.
			if !value.IsNil() && value.Elem().Kind() == reflect.Struct {
				nested, err := marshalStruct(value)
				if err != nil {
					return nil, err
				}
				mapped[tag] = nested
			} else {
				// Otherwise, assign the pointer value directly.
				mapped[tag] = value.Interface()
			}
		default:
			// For all other types, assign the value directly.
			mapped[tag] = value.Interface()
		}
	}
	return mapped, nil
}
