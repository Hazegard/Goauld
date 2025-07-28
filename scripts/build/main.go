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
	NoSeed         bool   `default:"false" help:"don't generate seed keys."`
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
		Id:          cfg.Id,
		Goos:        cfg.Goos,
		Goarch:      cfg.Goarch,
		Source:      pwd,
		EnvFile:     envFile,
		Output:      "output",
		Seed:        seed,
		ClientBuild: false,
	}

	err = cpl.Run()
	if err != nil {
		log.Error().Err(err).Msg("compiler.Compile()")
	}

}

func GenEnvFile(envFile string, cfg BuildConfig) (string, error) {
	newEnv := filepath.Join("output", fmt.Sprintf("%s.%s", envFile, time.Now().Format("2006-01-02T15:04:05")))
	err := compiler.CopyFile(envFile, newEnv)
	if err != nil {
		return "", err
	}

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
		content = compiler.ReplaceInFile(content, "SERVER__AGE_PRIVKEY", fmt.Sprintf("SERVER__AGE_PRIVKEY=%s", pubKey))
		content = compiler.ReplaceInFile(content, "AGENT__AGE_PUBKEY", fmt.Sprintf("AGENT__AGE_PUBKEY=%s", privKey))
	}

	if cfg.GenAccessToken {
		newToken, err := crypto.GeneratePassword(42)
		if err != nil {
			return "", err
		}
		content = compiler.ReplaceInFile(content, "SERVER__ACCESS_TOKEN", fmt.Sprintf("SERVER__ACCESS_TOKEN=%s", newToken))
	}

	return newEnv, os.WriteFile(newEnv, []byte(content), 0o700)
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
