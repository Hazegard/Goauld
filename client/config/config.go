package config

import (
	"Goauld/client/api"
	"Goauld/client/tui"
	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"
	"Goauld/common/utils"
	"fmt"
	"github.com/alecthomas/kong"
	"path/filepath"
	"strings"
)

var (
	_server       = "https://a.hazegard.fr"
	_access_token = "TODO_TOKEN"

	_verbosity = "0"

	defaultValues = kong.Vars{
		"_server":       _server,
		"_access_token": _access_token,

		"_verbosity": _verbosity,
	}
)

type ClientConfig struct {
	Server      string `default:"${_server}" short:"s" name:"server" optional:"" help:"HTTP Server to connect to."`
	AccessToken string `default:"${_access_token}" help:"Access token required to access the /manage/ endpoint."`

	Verbose int `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`

	Exec Exec `cmd:""`
	Tui  Tui  `cmd:""`
}

type Exec struct {
	Target string `arg:""`
}

func (e *Exec) Run(api *api.API, cfg ClientConfig) error {

	agent, err := api.GetAgentByName(e.Target)
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", agent)
	return nil
}

type Tui struct{}

func (t *Tui) Run(api *api.API, cfg ClientConfig) error {

	tt := tui.NewTui(api)
	return tt.Run()
}

func InitConfig() (*kong.Context, *ClientConfig, error) {
	cfg := &ClientConfig{}
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
		defaultValues,
	)

	log.SetLogLevel(cfg.Verbose)
	return app, cfg, nil
}
