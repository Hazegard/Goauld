package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"Goauld/common/log"
)

func main() {
	err := GenerateConfig("./agent/config/config.go", "agent_config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("generate agent config failed")
	}
	err = GenerateConfig("./server/config/config.go", "server_config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("generate server config failed")
	}
	err = GenerateConfig("./client/config.go", "client_config.yaml")
	if err != nil {
		log.Warn().Err(err).Msg("generate client config failed")
	}
}

func GenerateConfig(sourceFile string, configFile string) error {
	err, config := generateConfig(sourceFile)
	if err != nil {
		return err
	}
	return writeConfig(configFile, config)
}

func writeConfig(filename string, content string) error {
	return os.WriteFile(filename, []byte(content), 0o644)
}

func generateConfig(filename string) (error, string) {
	file, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	valMap := valMap(string(file))
	regexDefault := regexp.MustCompile(`default:"\${(?P<Default>.*?)}"`)
	regexHelp := regexp.MustCompile(`help:"(?P<Help>.*?)"`)
	regexName := regexp.MustCompile(`name:"(?P<Name>.*?)"`)

	content := extractContent(string(file), "type", "}")
	lines := strings.Split(content, "\n")
	output := ""
	for _, line := range lines {
		if line == "" {
			output += "\n"
			continue
		}
		defaultValueSlice := regexDefault.FindStringSubmatch(line)
		defaultValue := ""
		if len(defaultValueSlice) >= 2 {
			defaultValue = valMap[defaultValueSlice[1]]
		}
		helpSlice := regexHelp.FindStringSubmatch(line)
		if len(helpSlice) < 2 {
			return fmt.Errorf("error, no help found in line %s", line), ""
		}
		help := helpSlice[1]
		nameSlice := regexName.FindStringSubmatch(line)
		if len(nameSlice) < 2 {
			return fmt.Errorf("error, no name found in line %s", line), ""
		}
		name := nameSlice[1]
		output += fmt.Sprintf("# %+v\n%+v: %+v\n", help, name, valMap[defaultValue])
	}
	return nil, output
}

func valMap(c string) map[string]string {
	content := extractContent(c, "var", ")")
	valMap := make(map[string]string)
	for _, line := range strings.Split(content, "\n") {
		split := strings.Split(line, "=")
		if len(split) != 2 {
			continue
		}
		key := strings.TrimSpace(split[0])
		val := strings.Trim(strings.TrimSpace(split[1]), "\"")
		valMap[key] = val
	}
	return valMap
}

func extractContent(content string, start string, end string) string {
	split := strings.Split(content, "\n")
	doAdd := false
	res := []string{}
	for _, line := range split {
		if strings.HasPrefix(line, end) {
			doAdd = false
		}
		if doAdd {
			res = append(res, line)
		}
		if strings.HasPrefix(line, start) {
			doAdd = true
		}
	}
	return strings.Join(res, "\n")
}
