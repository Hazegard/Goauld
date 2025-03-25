package main

import (
	"Goauld/common/cli"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Goauld/client/api"
	"Goauld/client/common"
	"Goauld/client/tui"
	"Goauld/common/log"
	"Goauld/common/utils"

	"github.com/alecthomas/kong"
)

const APP_NAME = "tealc"

var (
	_server     = ""
	_ssh_server = ""

	_access_token = ""

	_verbosity = "0"
	_insecure  = "false"

	_generate_config = "false"
	_config_file     = ""

	_static_ssh_agent_map = ""
	_private_password     = ""

	_ssh_http             = "true"
	_ssh_socks            = "true"
	_ssh_local_socks_port = "1080"
	_ssh_local_http_port  = "3129"
	_ssh_ssh              = "true"
	_ssh_print            = "false"
	_ssh_proxy            = "false"

	_socks_socks            = "true"
	_socks_local_socks_port = "1080"
	_socks_ssh              = "false"
	_socks_print            = "false"
	_socks_proxy            = "false"

	_scp_print       = "false"
	_scp_source      = ""
	_scp_destination = ""
	_scp_args        = ""

	_pass_agent = ""
	_pass_type  = ""

	_compile_id               = "agent"
	_compile_goos             = ""
	_compile_goarch           = ""
	_compile_source           = ""
	_compile_env_file         = ""
	_compile_output           = ""
	_compile_drop_env         = "false"
	_compile_private_password = ""

	defaultValues = kong.Vars{
		"_server":       _server,
		"_ssh_server":   _ssh_server,
		"_access_token": _access_token,

		"_verbosity": _verbosity,
		"_insecure":  _insecure,

		"_generate_config": _generate_config,
		"_config_file":     _config_file,

		"_static_ssh_agent_map": _static_ssh_agent_map,

		"_ssh_http":             _ssh_http,
		"_ssh_socks":            _ssh_socks,
		"_ssh_local_socks_port": _ssh_local_socks_port,
		"_ssh_local_http_port":  _ssh_local_http_port,
		"_ssh_ssh":              _ssh_ssh,
		"_ssh_print":            _ssh_print,
		"_ssh_proxy":            _ssh_proxy,

		"_socks_socks":            _socks_socks,
		"_socks_local_socks_port": _socks_local_socks_port,
		"_socks_ssh":              _socks_ssh,
		"_socks_print":            _socks_print,
		"_socks_proxy":            _socks_proxy,

		"_scp_print":       _scp_print,
		"_scp_source":      _scp_source,
		"_scp_destination": _scp_destination,
		"_scp_args":        _scp_args,

		"_pass_agent": _pass_agent,
		"_pass_type":  _pass_type,

		"_compile_id":               _compile_id,
		"_compile_goos":             _compile_goos,
		"_compile_goarch":           _compile_goarch,
		"_compile_source":           _compile_source,
		"_compile_env_file":         _compile_env_file,
		"_compile_output":           _compile_output,
		"_compile_drop_env":         _compile_drop_env,
		"_compile_private_password": _compile_private_password,
	}
)

type ClientConfig struct {
	Server      string `default:"${_server}" short:"s" name:"server" optional:"" help:"HTTP Server to connect to."`
	AccessToken string `default:"${_access_token}" name:"access-token" help:"Access token required to access the /manage/ endpoint."`

	SshServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" optional:"" help:"SSH Server to connect to."`

	Verbose  int  `default:"${_verbosity}" help:"Verbosity. Repeat to increase" name:"verbose" short:"v" type:"counter"`
	Insecure bool `default:"${_insecure}" short:"k" name:"insecure" help:"Allow insecure connection (do not validate TLS certificate)."`

	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" help:"Generate configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" name:"config-file" optional:"" short:"c" help:"Configuration file to use."`

	AgentPassword   map[string]string `default:"${_static_ssh_agent_map}" name:"agent-password" help:"Agent password."`
	PrivatePassword string

	Ssh     Ssh      `cmd:"" name:"ssh" help:"Connect to the agent through SSH."`
	Socks   Socks    `cmd:"" name:"socks" help:"Mount the socks server exposed by the agent."`
	Scp     Scp      `cmd:"" name:"scp"  help:"Transfer files using SCP from/to the agent."`
	Tui     Tui      `cmd:"" name:"tui" help:"TUI used to manage the connected agents"`
	Pass    Password `cmd:"" default:"withargs" name:"pass"  help:"Retrieve the passwords used to connect to the agent."`
	Compile Compiler `cmd:"" name:"compile" help:"Compile the agent."`
}

func (c *ClientConfig) Validate() error {
	if c.AccessToken == "" {
		return errors.New("an access token is required (configuration file, environment variable, or --access-token)")
	}
	if c.Server == "" {
		return errors.New("a server URL is required (configuration file, environment variable, or --server)")
	}
	return nil
}

type Compiler struct{}

// GetSshdHost returns the configured sshd host
func (c *ClientConfig) GetSshdHost() string {
	split := strings.Split(c.SshServer, ":")
	return split[0]
}

// GetSshdPort returns the configured sshd port
func (c *ClientConfig) GetSshdPort() string {
	split := strings.Split(c.SshServer, ":")
	if len(split) == 2 {
		return split[1]
	}
	return ""
}

// ServerUrl returns control server URL
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

// GenerateYAMLConfig generate a yaml configuration associated to the currently running configuration
func (c *ClientConfig) GenerateYAMLConfig() (string, error) {
	return cli.GenerateYAMLWithComments(*c)
}

type Tui struct{}

// Run executes the tui subcommand
func (t *Tui) Run(api *api.API, cfg ClientConfig) error {
	tt := tui.NewTui(api)
	return tt.Run()
}

// InitConfig return the configuration depending on the command line arguments as well as the configuration files
func InitConfig() (*kong.Context, *ClientConfig, error) {
	cfgTmp := &ClientConfig{
		PrivatePassword: _private_password,
	}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfgTmp, err
	}
	configSearchDir := []string{
		filepath.Join(dir, fmt.Sprintf("%s.yaml", APP_NAME)),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", fmt.Sprintf("%s.yaml", APP_NAME))
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(strings.ToLower(APP_NAME)),
		kong.Description(common.Description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(APP_NAME)),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error {

			if ctx.Error == nil && ctx.Validate() == nil {
				fmt.Println(common.GetBanner())
				fmt.Println()
			}
			return kong.DefaultHelpPrinter(options, ctx)
		}),
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
