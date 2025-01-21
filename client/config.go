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
	_insecure  = "false"

	_exec_ssh   = "true"
	_exec_socks = "true"
	_exec_print = "false"
	_exec_proxy = "false"

	_local_socks_port = "1080"
	_local_sshd_port  = "22222"

	_generate_config = "false"

	defaultValues = kong.Vars{
		"_server":       _server,
		"_ssh_server":   _ssh_server,
		"_access_token": _access_token,

		"_verbosity": _verbosity,
		"_insecure":  _insecure,

		"_exec_ssh":   _exec_ssh,
		"_exec_socks": _exec_socks,
		"_exec_print": _exec_print,

		"_local_socks_port": _local_socks_port,
		"_local_sshd_port":  _local_sshd_port,

		"_generate_config": _generate_config,
	}
)

type ClientConfig struct {
	Server      string `default:"${_server}" short:"s" name:"server" optional:"" help:"HTTP Server to connect to."`
	AccessToken string `default:"${_access_token}" name:"access-token" help:"Access token required to access the /manage/ endpoint."`

	SshServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" optional:"" help:"SSH Server to connect to."`

	Verbose  int  `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`
	Insecure bool `default:"${_insecure}" short:"k" name:"insecure" help:"Allow insecure connection."`

	GenerateConfig bool   `default:"${_generate_config}" help:"Generate configuration file based on the current options."`
	ConfigFile     string `name:"config-file" type:"existingfile" optionnal:"" short:"c" help:"Configuration file to use."`

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

func (c *ClientConfig) ServerUrl() string {
	url := ""
	if strings.HasPrefix(c.Server, "http://") {
		url = c.Server
	} else if strings.HasPrefix(c.Server, "https://") {
		url = c.Server
	} else {
		url = "http://" + c.Server
	}

	return url
}

func (c *ClientConfig) GenerateYAMLConfig() (string, error) {
	return cli.GenerateYAMLWithComments(*c)
}

type Tui struct{}

func (t *Tui) Run(api *api.API, cfg ClientConfig) error {
	tt := tui.NewTui(api)
	return tt.Run()
}

type Password struct {
	Agent string   `name:"agent" help:"Agent to retrieve password."`
	Type  string   `name:"type" help:"Password to retrieve (OTP/Agent)."`
	Args  []string `arg:"" optional:""`
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
	cfgTmp := &ClientConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfgTmp, err
	}

	var kongOptions = []kong.Option{
		kong.Name(common.APP_NAME),
		kong.Description(common.Title("Client")),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, filepath.Join(dir, "client_config.yaml")),
		kong.DefaultEnvars(strings.ToUpper(common.APP_NAME)),
		defaultValues,
	}
	_ = kong.Parse(cfgTmp, kongOptions...)
	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar, cfgTmp.ConfigFile))

	}
	cfg := &ClientConfig{}
	app := kong.Parse(cfg, kongOptions...)

	log.SetLogLevel(cfg.Verbose)
	return app, cfg, nil
}
