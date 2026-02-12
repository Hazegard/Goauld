//go:build !mini

package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"

	"github.com/alecthomas/kong"
)

// Validate validates the configuration.
func (c *AgentConfig) Validate() error {
	var errs []error
	if c.OnlyWorkingDays {
		wd := NewWorkingDay(c.WorkingDayStart, c.WorkingDayEnd, c.WorkingDayTimeZone)
		errs = append(errs, wd.Validate())
	}
	if HasProto(c.TLSServer) {
		errs = append(errs, errors.New("the TLS server name must not contains protocol prefix"))
	}
	if HasProto(c.QuicServer) {
		errs = append(errs, errors.New("the QUIC server name must not contains protocol prefix"))
	}
	if len(c.PrivatePassword) > 72 {
		errs = append(errs, bcrypt.ErrPasswordTooLong)
	}

	if c.KillSwitch < 0 {
		errs = append(errs, errors.New("the kill-switch must not be negative"))
	}

	if !c.HTTP && c.MITMHTTP {
		errs = append(errs, errors.New("MITM HTTP proxy requires HTTP proxy to be enabled"))
	}

	return errors.Join(errs...)
}

// parse parses the command line arguments.
func parse() (*kong.Context, *AgentConfig, error) {
	cfgTmp := &AgentConfig{}
	dir, err := os.Getwd()
	if err != nil {
		return nil, cfgTmp, err
	}
	configSearchDir := []string{
		filepath.Join(dir, "goauld_agent.yaml"),
		filepath.Join(dir, "goauld.yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", "goauld_agent.yaml")
		configSearchDir = append(configSearchDir, homeConfig)
		homeConfig = filepath.Join(home, ".config", "goauld.yaml")
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(common.AppName()),
		kong.Description(common.Title(common.Appname) + "\n" + description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(common.AppName())),
		defaultValues,
	}
	_ = kong.Parse(cfgTmp, kongOptions...)

	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar([]string{}), cfgTmp.ConfigFile))
	}
	cfg := &AgentConfig{}
	app := kong.Parse(cfg, kongOptions...)

	if cfg.Quiet {
		log.SetLogLevel(-1)
	} else {
		log.SetLogLevel(cfg.Verbose)
	}

	return app, cfg, nil
}

// HasProto returns true if the url contains a protocol prefix.
func HasProto(u string) bool {
	split := strings.Split(u, "://")

	return len(split) > 1
}
