package compiler

import (
	"Goauld/common"
	"errors"
	"os"
	"os/exec"
)

// Goreleaser executes the Goreleaser build process with the provided compiler configuration.
func Goreleaser(cfg Compiler) error {
	c := []string{"goreleaser", "build", "--clean", "--auto-snapshot", "--skip=validate"}
	if cfg.Verbose > 0 {
		c = append(c, "--verbose")
	}
	// customBuild, err := DoSpecificBuild(cfg)
	// if err != nil {
	// 	return fmt.Errorf("error building: %s", err)
	// }
	var env []string
	if cfg.ID != "" && cfg.ID != "all" {
		c = append(c, "--id", cfg.ID)
	}
	if cfg.Goos != "" {
		env = append(env, "GOOS="+cfg.Goos)
	}
	if cfg.Goarch != "" {
		env = append(env, "GOARCH="+cfg.Goarch)
	}
	if cfg.Goos != "" || cfg.Goarch != "" {
		c = append(c, "--single-target")
	}
	if cfg.ClientBuild {
		env = append(env, "COMMIT="+common.Commit)
		env = append(env, "VERSION="+common.Version)
		env = append(env, "DATE="+common.Date)
	} else {
		env = append(env, "COMMIT=")
		env = append(env, "VERSION=")
		env = append(env, "DATE=")
	}
	if cfg.Compress {
		env = append(env, "COMPRESS=true")
	}
	if cfg.Tiny {
		env = append(env, "TINY=true")
	}
	if cfg.Literals {
		env = append(env, "LITERALS=true")
	}
	//nolint:gosec
	cmd := exec.Command(c[0], c[1:]...)

	_env, err := ParseEnvFile(cfg.EnvFile)

	_env = append(_env, env...)
	env = append(env, _env...)
	if err != nil {
		return err
	}
	cmd.Env = env
	cmd.Env = append(cmd.Env, os.Environ()...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Dir = cfg.Source

	return cmd.Run()
}

// DoSpecificBuild returns whether a specific build should be performed.
func DoSpecificBuild(cfg Compiler) (bool, error) {
	// All strings empty → return false.
	if cfg.ID == "" && cfg.Goos == "" && cfg.Goarch == "" {
		return false, nil
	}

	// All strings non-empty → return true.
	if cfg.ID != "" && cfg.Goos != "" && cfg.Goarch != "" {
		return true, nil
	}

	// Mixed values → return an error.
	return false, errors.New("error: mixed empty and non-empty values")
}
