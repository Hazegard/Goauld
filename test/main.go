package main

import (
	"encoding/json"
	"fmt"
	"github.com/alecthomas/kong"
	"io"
	"os"
	"strings"
)

var (
	test = "ldflag"
)

type Config struct {
	Test string `help:"Test"  name:"test" default:"${TEST}" env:"TEST"`
}

func main() {
	cli := Config{}
	readerconfig, err := os.Open("./agent.json")
	if err != nil {
		fmt.Println(err)
		return
	}
	jsonresolver, err := JSON(readerconfig)
	if err != nil {
		fmt.Println(err)
		return
	}

	kong.Parse(&cli,
		kong.Resolvers(jsonresolver),
		kong.Vars{"TEST": test})
	fmt.Println(cli.Test)
}

func JSON(r io.Reader) (kong.Resolver, error) {
	values := map[string]interface{}{}
	err := json.NewDecoder(r).Decode(&values)
	if err != nil {
		return nil, err
	}
	var f kong.ResolverFunc = func(context *kong.Context, parent *kong.Path, flag *kong.Flag) (interface{}, error) {
		for _, env := range flag.Envs {
			_, ok := os.LookupEnv(env)
			if ok {
				return nil, nil
			}
		}
		name := strings.ReplaceAll(flag.Name, "-", "_")
		snakeCaseName := snakeCase(flag.Name)
		raw, ok := values[name]
		if ok {
			return raw, nil
		} else if raw, ok = values[snakeCaseName]; ok {
			return raw, nil
		}
		raw = values
		for _, part := range strings.Split(name, ".") {
			if values, ok := raw.(map[string]interface{}); ok {
				raw, ok = values[part]
				if !ok {
					return nil, nil
				}
			} else {
				return nil, nil
			}
		}
		return raw, nil
	}

	return f, nil
}

func snakeCase(name string) string {
	name = strings.Join(strings.Split(strings.Title(name), "-"), "")
	return strings.ToLower(name[:1]) + name[1:]
}
