package main

import (
	"Goauld/common"
	"Goauld/common/log"
	"fmt"
	"os/exec"
	"strings"
)

type ClientConfig struct {
	GenAgeKey      string `default:"${true}" name:"server" optional:"" help:"generate Age keys."`
	GenAccessToken string `default:"${true}" help:"Generate the Access Token."`
}

var (
	requiredCommands = []string{
		"goreleaser",
		"age-keygen",
	}
)

func main() {
	missingCommands := checkCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", common.APP_NAME)).Str("commands", strings.Join(missingCommands, "\n")).Msg("Missing required commands")
	}

}

func checkCommands(cmds []string) []string {
	notFound := []string{}
	for _, cmd := range cmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			notFound = append(notFound, cmd)
		}

	}
	return notFound
}
