// Package main build
package main

import (
	"Goauld/client/compiler"
	"Goauld/common/crypto"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"Goauld/common"
	"Goauld/common/cli"

	"Goauld/common/log"

	"filippo.io/age"

	"github.com/alecthomas/kong"
)

type BuildConfig struct {
	GenAgeKey        bool   `default:"false" optional:"" help:"generate Age keys."`
	GenAccessToken   bool   `default:"false" help:"Generate the Access Token."`
	GenAgentPassword bool   `default:"true" help:"Generate the agent password."`
	ID               string `default:"" help:"[client|server|agent]."`
	Goos             string `default:"" help:"[darwin|linux|windows]."`
	Goarch           string `default:"" help:"[amd64|arm64|arm|386] (arm/386 only works for ID=client)."`
	NoSeed           bool   `default:"false" help:"don't generate seed keys."`
	NoPass           bool   `default:"false" help:"don't generate password."`
	Compress         bool   `default:"false" help:"Pack with UPX."`
	Verbose          int    `default:"0" name:"verbose" short:"v" type:"counter" help:"Verbosity of the logs. Repeat -v to increase"`
}

func main() {
	_, cfg, err := parse()
	if err != nil {
		log.Error().Err(err).Msg("failed to parse config")
	}

	err = compiler.MkdirAll("output")
	if err != nil {
		log.Error().Err(err).Msg("failed to create directory")
	}
	envFile, err := GenEnvFile(compiler.EnvFile, *cfg)
	if err != nil {
		log.Error().Err(err).Msg("genEnvFile()")

		return
	}
	pwd, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("os.Getwd()")

		return
	}

	seed := "__generate"
	if cfg.NoSeed {
		seed = ""
	}
	cpl := compiler.Compiler{
		ID:          cfg.ID,
		Goos:        cfg.Goos,
		Goarch:      cfg.Goarch,
		Source:      pwd,
		EnvFile:     envFile,
		Output:      "output",
		Seed:        seed,
		ClientBuild: false,
		NoPass:      !cfg.GenAgentPassword,
		Verbose:     cfg.Verbose,
		Compress:    cfg.Compress,
	}

	err = cpl.Run()
	if err != nil {
		log.Error().Err(err).Msg("compiler.Compile()")
		os.Exit(1)
	}
}

func GenEnvFile(envFile string, cfg BuildConfig) (string, error) {
	newEnv := filepath.Join("output", fmt.Sprintf("%s.%s", envFile, time.Now().Format("2006-01-02T15:04:05")))
	err := compiler.CopyFile(envFile, newEnv)
	if err != nil {
		return "", err
	}

	//nolint:gosec
	bytes, err := os.ReadFile(newEnv)
	if err != nil {
		return "", err
	}
	content := string(bytes)

	if cfg.GenAgeKey {
		pubKey, privKey, err := genAgeKey()
		if err != nil {
			return "", err
		}
		content = compiler.ReplaceInFile(content, "SERVER__AGE_PRIVKEY", "SERVER__AGE_PRIVKEY="+pubKey)
		content = compiler.ReplaceInFile(content, "AGENT__AGE_PUBKEY", "AGENT__AGE_PUBKEY="+privKey)
	}

	if cfg.GenAccessToken {
		newToken, err := crypto.GeneratePassword(42)
		if err != nil {
			return "", err
		}
		content = compiler.ReplaceInFile(content, "SERVER__ACCESS_TOKEN", "SERVER__ACCESS_TOKEN="+newToken)
	}

	return newEnv, os.WriteFile(newEnv, []byte(content), 0o600)
}

// parse parses the command line arguments.
func parse() (*kong.Context, *BuildConfig, error) {
	cfg := &BuildConfig{}
	dir, err := os.Getwd()
	if err != nil {
		return nil, cfg, err
	}
	app := kong.Parse(cfg,
		kong.Name(common.AppName()),
		kong.Description(common.Title("Build script")),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLOverwriteEnvVar([]string{}), filepath.Join(dir, strings.ToLower(common.AppName())+".yaml")),
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
