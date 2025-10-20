package compiler

import (
	sources "Goauld"
	"Goauld/client/common"
	goauldcommon "Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/rand"
	"embed"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/alecthomas/kong"
)

// Compiler holds the information used to compile the binaries.
type Compiler struct {
	ID            string `default:"${_compile_id}" help:"[client|server|agent]."`
	Goos          string `default:"${_compile_goos}" enum:",windows,linux,darwin" short:"O" help:"[darwin|linux|windows]."`
	Goarch        string `default:"${_compile_goarch}" enum:",amd64,arm64,arm,386" short:"A" help:"[amd64|arm64|arm|386] (arm/386 only works for ID=client)."`
	Source        string `default:"${_compile_source}" short:"s" help:"Source goauld directory."`
	EnvFile       string `default:"${_compile_env_file}" name:"env" help:"File containing environment variables."`
	Output        string `default:"${_compile_output}" short:"o" help:"Folder containing compiled agents."`
	Verbose       int    `default:"${_verbosity}" name:"verbose" short:"v" type:"counter" help:"Verbosity. Repeat to increase"`
	DropEnv       bool   `default:"${_compile_drop_env}" name:"drop-env" help:"Show then environment files required to compile the agent."`
	Seed          string `default:"${_compile_seed}" name:"seed" short:"S" help:"Seed to use to obfuscate agent."`
	AgentPassword string `default:"${_compile_private_password}" short:"p" help:"Static agent password."`
	ClientBuild   bool   `default:"true" hidden:"true"`
}

const (
	// EnvFile the default destination of created environment file.
	EnvFile = ".env.build"
)

var requiredCommands = []string{
	"go",
	"goreleaser",
	"garble",
}

// Run execute the compiler command.
func (c *Compiler) Run() error {
	if c.DropEnv {
		err := HandleDropEnv(sources.Sources)
		if err != nil {
			return fmt.Errorf("error reading embed .env file: %w", err)
		}

		return nil
	}
	log.Info().Msg("Compiler started")
	if c.Source == "" {
		tempDir, err := os.MkdirTemp("", "goauld_")
		if err != nil {
			return fmt.Errorf("could not create temp dir: %w", err)
		}
		defer func(path string) {
			err := os.RemoveAll(path)
			if err != nil {
				log.Error().Err(err).Str("Path", path).Msg("could not remove temp dir, please remove it manually")
			}
		}(tempDir)
		c.Source = tempDir
		err = drop(tempDir, sources.Sources)
		if err != nil {
			return fmt.Errorf("could not write files to temp dir: %w", err)
		}
		if c.EnvFile == "" {
			c.EnvFile = filepath.Join(tempDir, EnvFile+".tmpl")
		}
	}

	byteContent, err := os.ReadFile(c.EnvFile)
	if err != nil {
		return fmt.Errorf("could not read env file %s: %w", c.EnvFile, err)
	}
	content := string(byteContent)
	if c.AgentPassword != "" {
		content = ReplaceInFile(content, "AGENT__PRIVATE_PASSWORD=", "AGENT__PRIVATE_PASSWORD="+c.AgentPassword)
	}

	if c.Seed == "__generate" {
		seed, err := GenerateSecureRandomBase64(69)
		if err != nil {
			return fmt.Errorf("could not generate random seed: %w", err)
		}
		content = ReplaceInFile(content, "CLIENT__COMPILE_SEED=", "CLIENT__COMPILE_SEED="+seed)
	}
	err = MkdirAll(c.Output)
	if err != nil {
		return fmt.Errorf("could not create output directory %s: %w", c.Output, err)
	}
	newEnvFile := filepath.Join(c.Output, EnvFile)
	err = os.WriteFile(newEnvFile, []byte(content), 0o600)
	if err != nil {
		return fmt.Errorf("could not write env file %s: %w", newEnvFile, err)
	}
	if !c.ClientBuild {
		err = CopyFile(newEnvFile, c.EnvFile)
		if err != nil {
			return fmt.Errorf("could not override env file %s: %w", newEnvFile, err)
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
		return fmt.Errorf("compilation failed: %w", err)
	}

	return nil
}

// GenerateSecureRandomBase64 generates a secure random Base64 string of the given byte length.
func GenerateSecureRandomBase64(byteLength int) (string, error) {
	b := make([]byte, byteLength)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b), nil
}

// HandleDropEnv get the .env file stored in the embed struct and drops print the file to the stdout.
func HandleDropEnv(source embed.FS) error {
	fileContent, err := source.ReadFile(".env.build.tmpl")
	if err != nil {
		return err
	}
	//nolint:forbidigo
	fmt.Println(string(fileContent))

	return nil
}

// InitCompilerConfig returns the compiler configuration using the command line arguments as well
// as configuration files.
func InitCompilerConfig(appName string, defaultValues kong.Vars) (*kong.Context, *Compiler, error) {
	cfg := &Compiler{}
	dir, err := os.Getwd()
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
				//nolint:forbidigo
				fmt.Println(common.GetBanner())
				//nolint:forbidigo
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

// run execute the compiler.
func run(config Compiler) error {
	log.Info().Msgf("compiling %s", config.ID)
	missingCommands := CheckCommands(requiredCommands)
	if len(missingCommands) > 0 {
		log.Error().Err(fmt.Errorf("commands required to build %s", goauldcommon.Appname)).Str("commands", strings.Join(missingCommands, ", ")).Msg("Missing required commands")

		return fmt.Errorf("commands required to build %s", goauldcommon.Appname)
	}

	err := Goreleaser(config)
	if err != nil {
		log.Error().Err(err).Msg("error running goreleaser command")

		return fmt.Errorf("error running goreleaser command: %w", err)
	}

	artifacts, err := ParseArtifacts(filepath.Join(config.Source, "dist", "artifacts.json"))
	if err != nil {
		log.Error().Err(err).Msg("error parsing artifacts.json")

		return fmt.Errorf("error parsing artifacts.json: %w", err)
	}

	err = MoveArtifacts(artifacts, config.Source, config.Output)
	if err != nil {
		log.Error().Err(err).Msg("error updating artifacts")

		return fmt.Errorf("error updating artifacts: %w", err)
	}

	return nil
}

// drop writes to the destination directory the source files
// that will be used to compile the agent.
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

		// Handle vendored tarballs
		if strings.HasPrefix(path, "vendored/") && strings.HasSuffix(path, ".tar.gz") {
			tarDir := filepath.Join(destDir, strings.TrimSuffix(path, ".tar.gz"))
			log.Debug().Msgf("Extracting vendored archive: %s", path)
			if err := extractTarGz(tarDir, fileContent); err != nil {
				return err
			}
		}

		// Normal file extraction
		relativePath := strings.TrimPrefix(path, "")
		destPath := filepath.Join(destDir, relativePath)

		// Ensure parent dirs exist
		if err := os.MkdirAll(filepath.Dir(destPath), 0o750); err != nil {
			return err
		}

		// Write file
		if err := os.WriteFile(destPath, fileContent, 0o600); err != nil {
			return err
		}

		// Make garble scripts executable
		if strings.Contains(destPath, "scripts/garble") {
			_ = os.Chmod(destPath, 0o750)
		}

		log.Trace().Msgf("%s -> %s", path, destPath)

		return nil
	})

	return err
}

// extractTarGz extracts a .tar.gz archive from memory into destDir.
func extractTarGz(destDir string, data []byte) error {
	gzr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return err
		}

		base := filepath.Base(header.Name)
		if strings.HasPrefix(base, "._") || base == ".DS_Store" {
			continue
		}
		if strings.HasPrefix(header.Name, ".git") {
			continue
		}
		targetPath, err := SanitizeArchivePath(destDir, header.Name)
		if err != nil {
			log.Error().Err(err).Msgf("error extracting tar.gz from %s", header.Name)

			continue
		}
		//		targetPath := filepath.Join(destDir, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			//nolint:gosec
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o750); err != nil {
				return err
			}

			//nolint:gosec
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			//nolint:gosec
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()

				return err
			}
			outFile.Close()
		default:
			// Ignore symlinks and other types for safety
			log.Debug().Msgf("Skipping non-regular file in tar: %s", header.Name)
		}
	}

	return nil
}

// SanitizeArchivePath file pathing from "G305: Zip Slip vulnerability".
func SanitizeArchivePath(d string, t string) (string, error) {
	v := filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}
