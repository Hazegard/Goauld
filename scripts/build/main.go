package main

import (
	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/utils"
	"bufio"
	"encoding/json"
	"filippo.io/age"
	"fmt"
	"github.com/alecthomas/kong"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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
		"age-keygen",
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

	envFile, err := handleEnvFile(_envFile, *cfg)
	if err != nil {
		log.Error().Err(err).Msg("handleEnvFile()")
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

	err = updateArtifacts(artifacts)
	if err != nil {
		log.Error().Err(err).Msg("error updating artifacts")
		return
	}

}

type Artifact struct {
	Name         string `json:"name"`
	Path         string `json:"path"`
	InternalType int    `json:"internal_type"`
	Type         string `json:"type"`
	Goos         string `json:"goos,omitempty"`
	Goarch       string `json:"goarch,omitempty"`
	Goamd64      string `json:"goamd64,omitempty"`
	Target       string `json:"target,omitempty"`
	Extra        struct {
		Binary    string      `json:"Binary,omitempty"`
		Builder   string      `json:"Builder,omitempty"`
		Ext       string      `json:"Ext,omitempty"`
		ID        string      `json:"ID,omitempty"`
		Binaries  []string    `json:"Binaries,omitempty"`
		Checksum  string      `json:"Checksum,omitempty"`
		Format    string      `json:"Format,omitempty"`
		Replaces  interface{} `json:"Replaces"`
		WrappedIn string      `json:"WrappedIn,omitempty"`
	} `json:"extra,omitempty"`
	Go386   string `json:"go386,omitempty"`
	Goarm   string `json:"goarm,omitempty"`
	Goarm64 string `json:"goarm64,omitempty"`
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

func parseArtifacts(filePath string) ([]Artifact, error) {
	// Open the JSON file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %v", err)
	}
	defer file.Close()

	// Read the file's content
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	// Unmarshal the JSON data into the XXX struct
	var result []Artifact
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	// Return the unmarshaled struct
	return result, nil
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
		kong.Configuration(cli.YAML, filepath.Join(dir, strings.ToLower(common.APP_NAME)+".yaml")),
		kong.DefaultEnvars(strings.ToUpper(common.APP_NAME)),
		//defaultValues,
	)
	log.SetLogLevel(2)
	return app, cfg, nil
}

func handleEnvFile(envFile string, cfg BuildConfig) (string, error) {
	newEnv := fmt.Sprintf("%s.%s", envFile, time.Now().Format("2006-01-02T15:04:05"))
	err := copyFile(envFile, newEnv)
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
		content = replaceInFile(content, "SERVER__AGE_PRIVKEY", fmt.Sprintf("SERVER__AGE_PRIVKEY=%s", pubKey))
		content = replaceInFile(content, "AGENT__AGE_PUBKEY", fmt.Sprintf("AGENT__AGE_PUBKEY=%s", privKey))
	}

	if cfg.GenAccessToken {
		newtoken, err := crypto.GeneratePassword(42)
		if err != nil {
			return "", err
		}
		content = replaceInFile(content, "SERVER__ACCESS_TOKEN", fmt.Sprintf("SERVER__ACCESS_TOKEN=%s", newtoken))
	}

	return envFile, os.WriteFile(newEnv, []byte(content), 0700)
}

// ParseEnvFile reads an .env file and returns a slice of strings in "KEY=VALUE" format.
func ParseEnvFile(filepath string) ([]string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var envs []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		line = strings.TrimSpace(line)

		// Ignore comments and empty lines
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}

		// Ensure the line is in KEY=VALUE format
		if strings.Contains(line, "=") {
			envs = append(envs, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return envs, nil
}

func replaceInFile(content string, pattern string, new string) string {
	lines := strings.Split(content, "\n")
	newLines := []string{}
	for _, line := range lines {
		if strings.HasPrefix(line, pattern) {
			newLines = append(newLines, new)
		} else {
			newLines = append(newLines, line)
		}
	}
	return strings.Join(newLines, "\n")
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

func copyFile(src, dst string) error {
	// Open the source file for reading.
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	// Create the destination file for writing.
	destinationFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destinationFile.Close()

	// Copy the contents using io.Copy.
	_, err = io.Copy(destinationFile, sourceFile)
	if err != nil {
		return err
	}

	// Ensure that any writes to the destination file are committed.
	return destinationFile.Sync()
}

func MkdirAll(path string) error {
	return os.MkdirAll(path, 0755)
}

func updateArtifacts(artifacts []Artifact) error {
	for _, artifact := range artifacts {
		if artifact.Type == "Metadata" || artifact.Type == "Checksum" {
			continue
		}
		outDir := filepath.Join("output", artifact.Extra.ID)
		fmt.Println(outDir)
		err := MkdirAll(outDir)
		if err != nil {
			return fmt.Errorf("error creating dir: %v", err)
		}
		outFile := fmt.Sprintf("%s_%s-%s%s", artifact.Extra.Binary, artifact.Goos, artifact.Goarch, artifact.Extra.Ext)
		outPath := filepath.Join(outDir, outFile)
		err = copyFile(artifact.Path, outPath)
		if err != nil {
			return fmt.Errorf("error copying files: %v", err)
		}
	}
	return nil
}
