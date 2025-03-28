package compiler

import (
	"Goauld/common"
	"fmt"
	"os"
	"os/exec"
)

func Goreleaser(cfg Compiler) error {
	c := []string{"goreleaser", "build", "--clean", "--auto-snapshot", "--skip=validate"}
	// customBuild, err := DoSpecificBuild(cfg)
	// if err != nil {
	// 	return fmt.Errorf("error building: %s", err)
	// }
	var env []string
	if cfg.Id != "" && cfg.Id != "all" {
		c = append(c, "--id", cfg.Id) //, "--single-target")
	}
	if cfg.Goos != "" {
		env = append(env, "GOOS="+cfg.Goos)
	}
	if cfg.Goarch != "" {
		env = append(env, "GOARCH="+cfg.Goarch)
	}
	if cfg.Goos != "" && cfg.Goarch != "" {
		c = append(c, "--single-target")
	}
	if cfg.ClientBuild {
		env = append(env, fmt.Sprintf("COMMIT=%s", common.Commit))
		env = append(env, fmt.Sprintf("VERSION=%s", common.Version))
		env = append(env, fmt.Sprintf("DATE=%s", common.Date))
	} else {
		env = append(env, "COMMIT=")
		env = append(env, "VERSION=")
		env = append(env, "DATE=")
	}
	cmd := exec.Command(c[0], c[1:]...)

	_env, err := ParseEnvFile(cfg.EnvFile)

	env = append(env, _env...)
	if err != nil {
		return err
	}
	cmd.Env = append(cmd.Environ(), env...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = cfg.Source
	return cmd.Run()
}

// CheckCommands checks whether
func CheckCommands(cmds []string) []string {
	var notFound []string
	for _, cmd := range cmds {
		_, err := exec.LookPath(cmd)
		if err != nil {
			notFound = append(notFound, cmd)
		}

	}
	return notFound
}

// DoSpecificBuild returns whether a specific build should be performed
func DoSpecificBuild(cfg Compiler) (bool, error) {
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
