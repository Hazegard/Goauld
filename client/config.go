package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"Goauld/client/api"
	"Goauld/client/tui"
	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"
	"Goauld/common/utils"

	"github.com/alecthomas/kong"
)

var (
	_server       = "https://a.hazegard.fr"
	_ssh_server   = "localhost:2222"
	_access_token = "TODO_TOKEN"

	_verbosity = "0"

	_exec_ssh   = "true"
	_exec_socks = "true"
	_exec_print = "false"
	_exec_proxy = "false"

	_local_socks_port = "1080"
	_local_sshd_port  = "22222"

	defaultValues = kong.Vars{
		"_server":       _server,
		"_ssh_server":   _ssh_server,
		"_access_token": _access_token,

		"_verbosity": _verbosity,

		"_exec_ssh":   _exec_ssh,
		"_exec_socks": _exec_socks,
		"_exec_print": _exec_print,

		"_local_socks_port": _local_socks_port,
		"_local_sshd_port":  _local_sshd_port,
	}
)

type ClientConfig struct {
	Server      string `default:"${_server}" short:"s" name:"server" optional:"" help:"HTTP Server to connect to."`
	AccessToken string `default:"${_access_token}" help:"Access token required to access the /manage/ endpoint."`

	SshServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" optional:"" help:"SSH Server to connect to."`

	Verbose int `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`

	Exec Exec     `cmd:""`
	Tui  Tui      `cmd:""`
	Pass Password `cmd:"" default:"withargs"`
}

func (c *ClientConfig) GetSshdHost() string {
	split := strings.Split(c.SshServer, ":")
	return split[0]
}

func (c *ClientConfig) GetSshdPort() string {
	split := strings.Split(c.SshServer, ":")
	if len(split) == 2 {
		return split[1]
	}
	return ""
}

func (a *ClientConfig) ServerUrl() string {
	url := ""
	if strings.HasPrefix(a.Server, "http://") {
		url = a.Server
	} else if strings.HasPrefix(a.Server, "https://") {
		url = a.Server
	} else {
		url = "http://" + a.Server
	}

	return url
}

type Tui struct{}

func (t *Tui) Run(api *api.API, cfg ClientConfig) error {
	tt := tui.NewTui(api)
	return tt.Run()
}

type Password struct {
	Agent   string   `name:"agent" help:"Agent to retrieve password."`
	Type    string   `name:"type" help:"Password to retrieve (OTP/Agent)."`
	Garbage []string `arg:""`
}

func (p *Password) Run(api *api.API, cfg ClientConfig) error {
	agent, err := api.GetAgentByName(cfg.Pass.Agent)
	if err != nil {
		return err
	}
	switch cfg.Pass.Type {
	case "otp":
		fmt.Println(agent.OneTimePassword)
	case "agent":
		fmt.Println(agent.SshPasswd)
	}
	return nil
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
