package compiler

import (
	"Goauld"
	"Goauld/common"
	"Goauld/common/log"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

type Compiler struct {
	Id      string `default:"" help:"[client|server|agent]."`
	Goos    string `default:"" help:"[darwin|linux|windows]."`
	Goarch  string `default:"" help:"[amd64|arm64|arm|386] (arm/386 only works for Id=client)."`
	Source  string `default:"" help:"Source goa'uld directory."`
	EnvFile string `default:"" help:"File containing environment variables."`
	Output  string `default:"output" help:"File containing compiled compiled sources."`
}

const (
	artifactsFile = "./dist/artifacts.json"
	EnvFile       = ".env.build"
)

var requiredCommands = []string{
	"goreleaser",
}

func (c *Compiler) Run() error {
	log.Info().Msg("Compiler started")
	if c.Source == "" {
		tempDir, err := os.MkdirTemp("", "goauld_")
		if err != nil {
			return fmt.Errorf("could not create temp dir: %v", err)
		}
		// defer os.RemoveAll(tempDir)
		c.Source = tempDir
		err = drop(tempDir, Sources.Sources)
		if err != nil {
			return fmt.Errorf("could not write files to temp dir: %v", err)
		}
	}
	err := run(*c)
	if err != nil {
		return fmt.Errorf("compilation failed: %v", err)
	}

	return nil
}

func run(config Compiler) error {
	log.Info().Msgf("compiling %s", config.Id)
	missingCommands := CheckCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", common.App_Name)).Str("commands", strings.Join(missingCommands, "\n")).Msg("Missing required commands")
		return fmt.Errorf("commands required to build %s", common.App_Name)
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
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	fmt.Println(config.EnvFile)
	err = CopyFile(config.EnvFile, filepath.Join(config.Output, EnvFile))
	if err != nil {
		log.Error().Err(err).Msg("error copying env file")
		return fmt.Errorf("error copying env file: %v", err)
	}
	return nil
}

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

		log.Trace().Msgf("%s -> %s", path, destPath)
		return nil
	})

	return err
}
