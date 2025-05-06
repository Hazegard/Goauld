package compiler

import (
	"Goauld"
	"Goauld/client/common"
	goauldcommon "Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"
	"Goauld/common/utils"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"fmt"
	"github.com/alecthomas/kong"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Compiler holds the information used to compile the binaries
type Compiler struct {
	Id            string `default:"${_compile_id}" help:"[client|server|agent]."`
	Goos          string `default:"${_compile_goos}" short:"O" help:"[darwin|linux|windows]."`
	Goarch        string `default:"${_compile_goarch}" short:"A" help:"[amd64|arm64|arm|386] (arm/386 only works for Id=client)."`
	Source        string `default:"${_compile_source}" short:"s" help:"Source goauld directory."`
	EnvFile       string `default:"${_compile_env_file}" name:"env" help:"File containing environment variables."`
	Output        string `default:"${_compile_output}" short:"o" help:"Folder containing compiled compiled agents."`
	Verbose       int    `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`
	DropEnv       bool   `default:"${_compile_drop_env}" name:"drop-env" help:"Show then environment files required to compile the agent."`
	Seed          string `default:"${_compile_seed}" name:"seed" short:"S" help:"Seed to use to obfuscate agent."`
	AgentPassword string `default:"${_compile_private_password}" short:"p" help:"Static agent password."`
	ClientBuild   bool   `default:"true" hidden:"true"`
}

const (
	EnvFile = ".env.build"
)

var requiredCommands = []string{
	"go",
	"goreleaser",
	"garble",
}

// Run execute the compiler command
func (c *Compiler) Run() error {
	if c.DropEnv {
		err := HandleDropEnv(Sources.Sources)
		if err != nil {
			return fmt.Errorf("error reading embed .env file: %v", err)
		}
		return nil
	}
	log.Info().Msg("Compiler started")
	if c.Source == "" {
		tempDir, err := os.MkdirTemp("", "goauld_")
		if err != nil {
			return fmt.Errorf("could not create temp dir: %v", err)
		}
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				log.Error().Err(err).Str("Path", path).Msg("could not remove temp dir, please remove it manually")
			}
		}(tempDir)
		c.Source = tempDir
		err = drop(tempDir, Sources.Sources)
		if err != nil {
			return fmt.Errorf("could not write files to temp dir: %v", err)
		}
		if c.EnvFile == "" {
			c.EnvFile = filepath.Join(tempDir, EnvFile+".tmpl")
		}
	}

	byteContent, err := os.ReadFile(c.EnvFile)
	if err != nil {
		return fmt.Errorf("could not read env file %s: %v", c.EnvFile, err)
	}
	content := string(byteContent)
	if c.AgentPassword != "" {
		content = ReplaceInFile(content, "AGENT__PRIVATE_PASSWORD=", "AGENT__PRIVATE_PASSWORD="+c.AgentPassword)
	}

	if c.Seed == "__generate" {
		seed, err := GenerateSecureRandomBase64(69)
		if err != nil {
			return fmt.Errorf("could not generate random seed: %v", err)
		}
		content = ReplaceInFile(content, "CLIENT__COMPILE_SEED=", "CLIENT__COMPILE_SEED="+seed)
	}
	err = MkdirAll(c.Output)
	if err != nil {
		return fmt.Errorf("could not create output directory %s: %v", c.Output, err)
	}
	newEnvFile := filepath.Join(c.Output, EnvFile)
	err = os.WriteFile(newEnvFile, []byte(content), 0644)
	if err != nil {
		return fmt.Errorf("could not write env file %s: %v", newEnvFile, err)
	}
	if !c.ClientBuild {
		err = CopyFile(newEnvFile, c.EnvFile)
		if err != nil {
			return fmt.Errorf("could not override env file %s: %v", newEnvFile, err)
		}
	}
	c.EnvFile = newEnvFile
	// err = CopyFile(c.EnvFile, filepath.Join(c.Output, EnvFile))
	// if err != nil {
	// 	log.Error().Err(err).Msg("error copying env file")
	// 	return fmt.Errorf("error copying env file: %v", err)
	// }

	err = run(*c)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	return nil
}

// GenerateSecureRandomBase64 generates a secure random Base64 string of the given byte length
func GenerateSecureRandomBase64(byteLength int) (string, error) {
	bytes := make([]byte, byteLength)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(bytes), nil
}

// HandleDropEnv get the .env file stored in the embed struct and drops print the file to the stdout
func HandleDropEnv(source embed.FS) error {
	fileContent, err := source.ReadFile(".env.build.tmpl")
	if err != nil {
		return err
	}
	fmt.Println(string(fileContent))
	return nil
}

// InitCompilerConfig returns the compiler configuration using the command line arguments as well
// as configuration files
func InitCompilerConfig(appName string, defaultValues kong.Vars) (*kong.Context, *Compiler, error) {
	cfg := &Compiler{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfg, err
	}
	configSearchDir := []string{
		filepath.Join(dir, "client_config.yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", strings.ToLower(appName), "client_config.yaml")
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(strings.ToLower(appName)),
		kong.Description(common.Description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(appName)),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error {
			if ctx.Error == nil {
				fmt.Println(common.GetBanner())
				fmt.Println()
			}
			return kong.DefaultHelpPrinter(options, ctx)
		}),
		defaultValues,
	}
	app := kong.Parse(cfg, kongOptions...)

	log.SetLogLevel(cfg.Verbose)
	return app, cfg, nil
}

// run execute the compiler
func run(config Compiler) error {
	log.Info().Msgf("compiling %s", config.Id)
	missingCommands := CheckCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", goauldcommon.App_Name)).Str("commands", strings.Join(missingCommands, ", ")).Msg("Missing required commands")
		return fmt.Errorf("commands required to build %s", goauldcommon.App_Name)
	}

	err := Goreleaser(config)
	if err != nil {
		log.Error().Err(err).Msg("error running goreleaser command")
		return fmt.Errorf("error running goreleaser command: %v", err)
	}

	artifacts, err := ParseArtifacts(filepath.Join(config.Source, "dist", "artifacts.json"))
	if err != nil {
		log.Error().Err(err).Msg("error parsing artifacts.json")
		return fmt.Errorf("error parsing artifacts.json: %v", err)
	}

	err = MoveArtifacts(artifacts, config.Source, config.Output)
	if err != nil {
		log.Error().Err(err).Msg("error updating artifacts")
		return fmt.Errorf("error updating artifacts: %v", err)
	}
	return nil
}

// drop writes to the destination directory the source files
// that will be used to compile the agent
func drop(destDir string, source embed.FS) error {
	// Walk through all files and directories in the embedded content
	err := fs.WalkDir(source, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip directories, we want to extract files
		if d.IsDir() {
			return nil
		}

		// Get the file content from the embedded FS
		fileContent, err := source.ReadFile(path)
		if err != nil {
			return err
		}

		// Create the target file path
		// Remove the 'files/' part from the embedded path
		relativePath := strings.TrimPrefix(path, "")
		destPath := filepath.Join(destDir, relativePath)

		// Create the necessary directories
		dir := filepath.Dir(destPath)
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}

		// Write the content to the file
		err = os.WriteFile(destPath, fileContent, 0644)
		if err != nil {
			return err
		}

		if strings.Contains(destPath, "scripts/garble") {
			err = os.Chmod(destPath, 0755)
			if err != nil {
				return err
			}
		}

		log.Trace().Msgf("%s -> %s", path, destPath)
		return nil
	})

	return err
}
