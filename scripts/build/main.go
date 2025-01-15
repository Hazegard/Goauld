package main

import (
	"Goauld/common"
	"Goauld/common/cli"

	"Goauld/common/log"
	"Goauld/common/utils"
	"filippo.io/age"
	"fmt"
	"github.com/alecthomas/kong"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BuildConfig struct {
	GenAgeKey      bool `default:"true" optional:"" help:"generate Age keys."`
	GenAccessToken bool `default:"true" help:"Generate the Access Token."`
}

const (
	artifactsFile = "./dist/artifacts.json"
	_envFile      = ".env.build"
)

var (
	requiredCommands = []string{
		"goreleaser",
	}
)

func main() {
	_, cfg, err := parse()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse config")
	}
	missingCommands := checkCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", common.APP_NAME)).Str("commands", strings.Join(missingCommands, "\n")).Msg("Missing required commands")
		return
	}

	envFile, err := GenEnvFile(_envFile, *cfg)
	if err != nil {
		log.Error().Err(err).Msg("genEnvFile()")
		return
	}
	err = goreleaser(envFile)
	if err != nil {
		log.Error().Err(err).Msg("error running goreleaser command")
		return
	}

	artifacts, err := parseArtifacts(artifactsFile)
	if err != nil {
		log.Error().Err(err).Msg("error parsing artifacts.json")
	}

	err = moveArtifacts(artifacts)
	if err != nil {
		log.Error().Err(err).Msg("error updating artifacts")
		return
	}

}

func goreleaser(envFile string) error {
	cmd := exec.Command("goreleaser", "release", "--clean", "--skip", "publish", "--snapshot")
	env, err := ParseEnvFile(envFile)
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Environ(), env...)

	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func checkCommands(cmds []string) []string {
	var notFound []string
	for _, cmd := range cmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			notFound = append(notFound, cmd)
		}

	}
	return notFound
}

// parse parses the command line arguments
func parse() (*kong.Context, *BuildConfig, error) {
	cfg := &BuildConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfg, err
	}
	app := kong.Parse(cfg,
		kong.Name(common.APP_NAME),
		kong.Description("TODO"),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLOverwriteEnvVar, filepath.Join(dir, strings.ToLower(common.APP_NAME)+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(common.APP_NAME)),
		//defaultValues,
	)
	log.SetLogLevel(2)
	return app, cfg, nil
}

func genAgeKey() (string, string, error) {
	key, err := age.GenerateX25519Identity()
	if err != nil {
		return "", "", err
	}
	pubkey := key.Recipient().String()
	privkey := key.String()
	return pubkey, privkey, err
}
