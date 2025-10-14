package main

import (
	"Goauld/client/types"
	common2 "Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/yaml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"Goauld/client/api"
	"Goauld/client/common"
	"Goauld/client/tui"
	"Goauld/common/log"

	"github.com/alecthomas/kong"
)

const AppName = "tealc"

var (
	_server     = ""
	_ssh_server = "" //nolint:revive

	_access_token = "" //nolint:revive
	_admin_token  = "" //nolint:revive

	_verbosity = "0"
	_insecure  = "false"

	_version         = "false"
	_generate_config = "false" //nolint:revive
	_config_file     = ""      //nolint:revive

	_static_ssh_agent_map   = ""      //nolint:revive
	_private_password       = ""      //nolint:revive
	_prompt_static_password = "false" //nolint:revive
	_save_static_password   = "false" //nolint:revive

	_ssh_http             = "true"  //nolint:revive
	_ssh_socks            = "true"  //nolint:revive
	_ssh_local_socks_port = "1080"  //nolint:revive
	_ssh_local_http_port  = "3128"  //nolint:revive
	_ssh_ssh              = "true"  //nolint:revive
	_ssh_print            = "false" //nolint:revive
	_ssh_proxy            = "false" //nolint:revive

	_socks_socks            = "true"  //nolint:revive
	_socks_local_socks_port = "1080"  //nolint:revive
	_socks_ssh              = "false" //nolint:revive
	_socks_print            = "false" //nolint:revive
	_socks_proxy            = "false" //nolint:revive

	_scp_print       = "false" //nolint:revive
	_scp_source      = ""      //nolint:revive
	_scp_destination = ""      //nolint:revive
	_scp_args        = ""      //nolint:revive

	_pass_agent = "" //nolint:revive
	_pass_type  = "" //nolint:revive

	_compile_id               = "agent"      //nolint:revive
	_compile_goos             = ""           //nolint:revive
	_compile_goarch           = ""           //nolint:revive
	_compile_source           = ""           //nolint:revive
	_compile_env_file         = ""           //nolint:revive
	_compile_output           = "output"     //nolint:revive
	_compile_drop_env         = "false"      //nolint:revive
	_compile_seed             = "__generate" //nolint:revive
	_compile_private_password = ""           //nolint:revive

	defaultValues = kong.Vars{
		"_server":       _server,
		"_ssh_server":   _ssh_server,
		"_access_token": _access_token,
		"_admin_token":  _admin_token,

		"_verbosity": _verbosity,
		"_insecure":  _insecure,

		"_version":         _version,
		"_generate_config": _generate_config,
		"_config_file":     _config_file,

		"_static_ssh_agent_map":   _static_ssh_agent_map,
		"_private_password":       "",
		"_prompt_static_password": _prompt_static_password,
		"_save_static_password":   _save_static_password,

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
		"_compile_seed":             _compile_seed,
		"_compile_private_password": _compile_private_password,
	}
)

type ClientConfig struct {
	Server    string `default:"${_server}" short:"s" name:"server" yaml:"server"  optional:"" help:"HTTP Server to connect to."`
	SSHServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" yaml:"ssh-server" optional:"" help:"SSH Server to connect to."`

	AccessToken string `default:"${_access_token}" name:"access-token" yaml:"access-token" help:"Access token required to access the /manage/ endpoint."`
	AdminToken  string `default:"${_admin_token}" name:"admin-token" yaml:"admin-token" help:"Admin token required to access the /admin/ endpoint."`

	Verbose  int  `default:"${_verbosity}" name:"verbose" yaml:"verbose"  short:"v" type:"counter" help:"Verbosity. Repeat to increase"`
	Insecure bool `default:"${_insecure}" short:"k" name:"insecure" yaml:"insecure" help:"Allow insecure connection (do not validate TLS certificate)."`

	Version        bool   `default:"${_version}" name:"version" yaml:"version" short:"V" help:"Show version information"`
	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" yaml:"generate-config" help:"Generate configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" name:"config-file" yaml:"config-file" optional:"" short:"c" help:"Configuration file to use."`

	AgentPassword   map[string]string `default:"${_static_ssh_agent_map}" hidden:"true" name:"agent-password" yaml:"agent-password" help:"Agent password map."`
	PrivatePassword string            `default:"" name:"password" yaml:"password" short:"P" help:"Agent password."`
	PromptPassword  bool              `default:"${_prompt_static_password}" name:"prompt" yaml:"prompt" short:"Q" help:"Prompt for the agent private password."`
	SavePassword    bool              `default:"${_save_static_password}" name:"save" yaml:"save" negatable:"" help:"Save the prompted password in the config file."`

	SSH     SSH      `cmd:"" name:"ssh" help:"Connect to the agent through SSH."`
	Socks   Socks    `cmd:"" name:"socks" help:"Mount the socks server exposed by the agent."`
	SCP     Scp      `cmd:"" name:"scp" help:"Transfer files using SCP from/to the agent."`
	TUI     Tui      `cmd:"" name:"tui" help:"TUI used to manage the connected agents"`
	Pass    Password `cmd:"" default:"withargs" name:"pass"  help:"Retrieve the passwords used to connect to the agent."`
	Compile Compiler `cmd:"" name:"compile" help:"Compile the agent."`
	Admin   Admin    `cmd:"" name:"admin" help:"Admin command." hidden:"true"`
	VsCode  VsCode   `cmd:"" name:"vscode" help:"Start vsCode in remote mode."`

	SearchConfigDir string `hidden:""`
}

// ValidateConfig check the required options and returns an error if they are not set.
func (cfg *ClientConfig) ValidateConfig() error {
	if cfg.GenerateConfig {
		// we do not validate the current config as we might want to have the access token or server empty
		return nil
	}
	if cfg.AccessToken == "" {
		return errors.New("an access token is required (configuration file, environment variable, or --access-token)")
	}
	if cfg.Server == "" {
		return errors.New("a server URL is required (configuration file, environment variable, or --server)")
	}

	return nil
}

type Compiler struct {
	Args []string `hidden:"true" arg:"" passthrough:""`
}

// GetSshdHost returns the configured sshd host.
func (cfg *ClientConfig) GetSshdHost() string {
	split := strings.Split(cfg.SSHServer, ":")

	return split[0]
}

// GetSshdPort returns the configured sshd port.
func (cfg *ClientConfig) GetSshdPort() string {
	split := strings.Split(cfg.SSHServer, ":")
	if len(split) == 2 {
		return split[1]
	}

	return ""
}

// ServerURL returns control server URL.
func (cfg *ClientConfig) ServerURL() string {
	var url string
	switch {
	case strings.HasPrefix(cfg.Server, "http://"):
		url = cfg.Server
	case strings.HasPrefix(cfg.Server, "https://"):
		url = cfg.Server
	default:
		url = "http://" + cfg.Server
	}

	return url
}

// GenerateYAMLConfig generate a YAML configuration associated with the currently running configuration.
func (cfg *ClientConfig) GenerateYAMLConfig() (string, error) {
	return cli.GenerateYAMLWithComments(*cfg)
}

// IsFlagInCommandLine checks whether the flag is provided in the command line arguments.
func (cfg *ClientConfig) IsFlagInCommandLine(long string, short string) bool {
	for _, field := range os.Args {
		if (long != "" && field == long) || (short != "" && field == short) {
			return true
		}
	}

	return false
}

type Tui struct{}

// Run executes the tui subcommand.
func (t *Tui) Run(api *api.API, cfg ClientConfig) error {
	tt := tui.NewTui(api, cfg.AgentPassword)
	agent, mode, err := tt.Run()

	if err != nil {
		return err
	}
	if agent == "" {
		return nil
	}
	switch mode {
	case "ssh":
		cfg.SSH.Target = agent

		return cfg.SSH.Run(api, cfg)
	case "vscode":
		cfg.VsCode.Target = agent

		return cfg.VsCode.Run(api, cfg)
	}

	return nil
}

// InitConfig return the configuration depending on the command line arguments as well as the configuration files.
func InitConfig() (*kong.Context, *ClientConfig, *kong.Context, error) {
	cfgTmp := &ClientConfig{
		PrivatePassword: _private_password,
	}
	dir, err := os.Getwd()
	if err != nil {
		return nil, cfgTmp, nil, err
	}
	configSearchDir := []string{
		filepath.Join(dir, AppName+".yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", AppName+".yaml")
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(strings.ToLower(AppName)),
		kong.Description(common2.Title(AppName) + "\n" + common.Description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(AppName)),
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
	ctx := kong.Parse(cfgTmp, kongOptions...)
	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar([]string{"password"}), cfgTmp.ConfigFile))
		configSearchDir = append([]string{cfgTmp.ConfigFile}, configSearchDir...)
	}
	cfg := &ClientConfig{}
	app := kong.Parse(cfg, kongOptions...)

	if cfgTmp.ConfigFile != "" {
		abs, err := filepath.Abs(cfgTmp.ConfigFile)
		if err != nil {
			cfg.ConfigFile = cfgTmp.ConfigFile
		} else {
			cfg.ConfigFile = abs
		}
	}
	cfg.SearchConfigDir = cli.GetConfigFile(configSearchDir...)

	log.SetLogLevel(cfg.Verbose)

	return app, cfg, ctx, cfg.ValidateConfig()
}

// Target returns the target by parsing the subcommands to find which one is currently executed.
func (cfg *ClientConfig) Target() string {
	if cfg.SSH.Target != "" {
		return cfg.SSH.Target
	}
	if cfg.SCP.Target != "" {
		return cfg.SCP.Target
	}
	if cfg.Socks.Target != "" {
		return cfg.Socks.Target
	}

	return ""
}

// UpdatePassConfigFile updates the configuration file to set the new static password.
func (cfg *ClientConfig) UpdatePassConfigFile() error {
	if cfg.SavePassword && cfg.PrivatePassword != "" {
		return yaml.UpdateAgentPasswordConfig(cfg.SearchConfigDir, cfg.Target(), cfg.PrivatePassword)
	}

	return nil
}

// ShouldPrompt return whether the client should display a prompt.
func (cfg *ClientConfig) ShouldPrompt(agent types.Agent) bool {
	if cfg.PromptPassword {
		return true
	}
	if cfg.PrivatePassword != "" {
		return false
	}
	_, ok := cfg.AgentPassword[cfg.Target()]
	if ok {
		return false
	}

	return agent.HasStaticPassword
}

// Prompt prompts the password to the user.
func (cfg *ClientConfig) Prompt(agent string) error {
	pass, err := tui.Prompt(agent)
	if err != nil {
		return err
	}
	cfg.PromptPassword = true
	cfg.PrivatePassword = pass

	return nil
}
