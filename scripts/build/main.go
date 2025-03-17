package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"Goauld/common"
	"Goauld/common/cli"

	"Goauld/common/log"
	"Goauld/common/utils"
	"filippo.io/age"

	"github.com/alecthomas/kong"
)

type BuildConfig struct {
	GenAgeKey      bool   `default:"false" optional:"" help:"generate Age keys."`
	GenAccessToken bool   `default:"false" help:"Generate the Access Token."`
	Id             string `default:"" help:"[client|server|agent]."`
	Goos           string `default:"" help:"[darwin|linux|windows]."`
	Goarch         string `default:"" help:"[amd64|arm64|arm|386] (arm/386 only works for Id=client)."`
}

const (
	artifactsFile = "./dist/artifacts.json"
	_envFile      = ".env.build"
)

var requiredCommands = []string{
	"goreleaser",
}

func main() {
	_, cfg, err := parse()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse config")
	}
	missingCommands := checkCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", common.App_Name)).Str("commands", strings.Join(missingCommands, "\n")).Msg("Missing required commands")
		return
	}

	err = MkdirAll("output")
	if err != nil {
		log.Error().Err(err).Msg("failed to create directory")
	}
	envFile, err := GenEnvFile(_envFile, *cfg)
	if err != nil {
		log.Error().Err(err).Msg("genEnvFile()")
		return
	}
	err = goreleaser(*cfg, envFile)
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
	err = copyFile(envFile, filepath.Join("output", _envFile))
	if err != nil {
		log.Error().Err(err).Msg("error copying env file")
	}
}

func goreleaser(cfg BuildConfig, envFile string) error {
	c := []string{"goreleaser", "build", "--clean", "--auto-snapshot", "--skip=validate"}
	customBuild, err := DoSpecificBuild(cfg)
	if err != nil {
		return fmt.Errorf("error building: %s", err)
	}
	var env []string
	if customBuild {
		c = append(c, "--id", cfg.Id, "--single-target")
		env = append(env, "GOOS="+cfg.Goos)
		env = append(env, "GOARCH="+cfg.Goarch)
	}
	cmd := exec.Command(c[0], c[1:]...)

	_env, err := ParseEnvFile(envFile)
	env = append(env, _env...)
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
		kong.Name(common.AppName()),
		kong.Description(common.Title("Build script")),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLOverwriteEnvVar, filepath.Join(dir, strings.ToLower(common.AppName())+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(common.AppName())),
		// defaultValues,
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

func DoSpecificBuild(cfg BuildConfig) (bool, error) {
	// All strings empty → return false.
	if cfg.Id == "" && cfg.Goos == "" && cfg.Goarch == "" {
		return false, nil
	}

	// All strings non-empty → return true.
	if cfg.Id != "" && cfg.Goos != "" && cfg.Goarch != "" {
		return true, nil
	}

	// Mixed values → return an error.
	return false, fmt.Errorf("error: mixed empty and non-empty values")
}
