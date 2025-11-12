package main

import (
	"Goauld/client/compiler"
	"Goauld/client/types"
	common2 "Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/yaml"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

	_quiet     = "false"
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
	_ssh_wg               = "true"  //nolint:revive
	_ssh_socks            = "true"  //nolint:revive
	_ssh_local_socks_port = "1080"  //nolint:revive
	_ssh_local_http_port  = "3128"  //nolint:revive
	_wg_port              = "51820" //nolint:revive
	_ssh_ssh              = "true"  //nolint:revive
	_ssh_print            = "false" //nolint:revive
	_ssh_proxy            = "false" //nolint:revive
	_ssh_log              = "false" //nolint:revive

	_socks_socks            = "true"  //nolint:revive
	_socks_local_socks_port = "1080"  //nolint:revive
	_socks_ssh              = "false" //nolint:revive
	_socks_print            = "false" //nolint:revive
	_socks_proxy            = "false" //nolint:revive

	_scp_print       = "false" //nolint:revive
	_scp_log         = "false" //nolint:revive
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
	_compile_nopass           = "false"      //nolint:revive
	_compile_compress         = "false"      //nolint:revive

	defaultValues = kong.Vars{
		"AppName":       AppName,
		"_server":       _server,
		"_ssh_server":   _ssh_server,
		"_access_token": _access_token,
		"_admin_token":  _admin_token,

		"_quiet":     _quiet,
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
		"_ssh_wg":               _ssh_wg,
		"_ssh_socks":            _ssh_socks,
		"_ssh_local_socks_port": _ssh_local_socks_port,
		"_ssh_local_http_port":  _ssh_local_http_port,
		"_wg_port":              _wg_port,
		"_ssh_ssh":              _ssh_ssh,
		"_ssh_print":            _ssh_print,
		"_ssh_proxy":            _ssh_proxy,
		"_ssh_log":              _ssh_log,

		"_socks_socks":            _socks_socks,
		"_socks_local_socks_port": _socks_local_socks_port,
		"_socks_ssh":              _socks_ssh,
		"_socks_print":            _socks_print,
		"_socks_proxy":            _socks_proxy,

		"_scp_print":       _scp_print,
		"_scp_log":         _scp_log,
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
		"_compile_nopass":           _compile_nopass,
		"_compile_compress":         _compile_compress,
	}
)

type ClientConfig struct {
	Server    string `default:"${_server}" short:"s" name:"server" yaml:"server" optional:"" help:"HTTP server address to connect to."`
	SSHServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" yaml:"ssh-server" optional:"" help:"SSH server address to connect to."`

	AccessToken string `default:"${_access_token}" name:"access-token" yaml:"access-token" help:"Access token for the /manage/ API endpoint."`
	AdminToken  string `default:"${_admin_token}" name:"admin-token" yaml:"admin-token" help:"Admin token for the /admin/ API endpoint."`

	Quiet    bool `default:"${_quiet}" short:"q" name:"quiet" yaml:"quiet" short:"q" help:"Suppress all log output."`
	Verbose  int  `default:"${_verbosity}" short:"v" name:"verbose" yaml:"verbose" short:"v" type:"counter" help:"Increase verbosity level. Repeat for more detailed logs."`
	Insecure bool `default:"${_insecure}" short:"k" name:"insecure" yaml:"insecure" help:"Allow insecure connections (skip TLS certificate verification)."`

	Version        bool   `default:"${_version}" short:"V" name:"version" yaml:"version" short:"V" help:"Display version information and exit."`
	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" yaml:"generate-config" help:"Generate a configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" short:"c" name:"config-file" yaml:"config-file" optional:"" short:"c" help:"Path to configuration file."`

	AgentPassword   map[string]string `default:"${_static_ssh_agent_map}" hidden:"true" name:"agent-password" yaml:"agent-password" help:"Agent password map (internal use only)."`
	PrivatePassword string            `default:"" short:"P" name:"password" yaml:"password" short:"P" help:"Agent private password."`
	PromptPassword  bool              `default:"${_prompt_static_password}" short:"Q" name:"prompt" yaml:"prompt" short:"Q" yaml:"prompt" help:"Prompt for the agent's private password."`
	SavePassword    bool              `default:"${_save_static_password}" name:"save" yaml:"save" negatable:"" help:"Save the prompted password in the configuration file."`

	SSH       SSH       `cmd:"" name:"ssh" yaml:"ssh" help:"Connect to an agent via SSH.\n This command supports SSH cli flags, however, these flags must be placed at the end of the command line, e.g:\n $ ${AppName} ssh [TEALC OPTIONS] AGENT [SSH OPTIONS]"`
	Socks     Socks     `cmd:"" name:"socks" yaml:"socks" help:"Expose the SOCKS proxy provided by the agent.\n $ ${AppName} socks [TEALC OPTIONS] AGENT"`
	SCP       Scp       `cmd:"" name:"scp" yaml:"scp" help:"Transfer files to/from the agent using SCP.\n- AGENT=>Client:\n $ ${AppName} scp AGENT:/remote/path /local/path\n- CLIENT=>AGENT:\n $ ${AppName} scp /local/path AGENT:/remote/path\n/!\\ On windows agent, the path must use \"/\" instead of \"\\\"\n - $ ${AppName} scp AGENT:C:/remote/path /local/path"`
	Rsync     Rsync     `cmd:"" name:"rsync" yaml:"rsync" help:"Transfer files to/from the agent using Rsync.\n- AGENT=>Client:\n $ ${AppName} rsync AGENT:/remote/path /local/path\n- CLIENT=>AGENT:\n $ ${AppName} rsync /local/path AGENT:/remote/path\n/!\\ On windows agents, the path must use \"/\" instead of \"\\\"\n - $ ${AppName} scp AGENT:C:/remote/path /local/path"`
	Jump      Jump      `cmd:"" name:"jump" yaml:"jump" help:"SSH into a remote host using the agent as a jump server (similar to ssh -J option).\nIt supports arbitrary SSH flags (SSH keys, etc.) placed at the end of the command line.\n ${AppName} jump AGENT REMOTE_HOST [SSH_OPTIONS]"`
	VsCode    VsCode    `cmd:"" name:"vscode" yaml:"vscode" help:"Launch VS Code in remote mode via the agent.\n /!\\ It downloads and executes the VSCode remote server on the agent in the folder executing the agent, so it may trigger events.\n Moreover, the cleaning system in place that should delete the VSCode folder when the agent exits might not work properly.\n So it is required to manually cleans the folder."`
	Clipboard Clipboard `cmd:"" name:"clip" yaml:"clip" help:"Access or modify the agent clipboard."`
	TUI       Tui       `cmd:"" name:"tui" yaml:"tui" help:"Launch the text-based interface for managing connected agents."`
	Pass      Password  `cmd:"" default:"withargs" name:"pass" yaml:"pass" help:"Retrieve stored passwords used by the agent."`
	Kill      Kill      `cmd:"" name:"kill" yaml:"kill" help:"Terminate a running agent."`
	Reset     Reset     `cmd:"" name:"reset" yaml:"reset" help:"Reset an agent to its default state."`
	Delete    Delete    `cmd:"" name:"delete" yaml:"delete" help:"Delete an agent permanently."`
	List      List      `cmd:"" name:"list" yaml:"list" help:"List all available agents."`
	Compile   Compiler  `cmd:"" name:"compile" yaml:"compile" help:"Compile a new agent binary.\n $ ${AppName} compile -O windows -A amd64"`
	Admin     Admin     `cmd:"" name:"admin" yaml:"admin" help:"Administrative commands (internal use)." hidden:"true"`
	Wireguard Wireguard `cmd:"" name:"wireguard" yaml:"wireguard" help:"Generate or manage WireGuard configuration."`

	SearchConfigDir string `hidden:""`
}

func (cfg *ClientConfig) EnvVar(target string) []string {
	var env []string
	env = append(env, prefixEnv("SERVER", cfg.Server))
	env = append(env, prefixEnv("SSH_SERVER", cfg.SSHServer))

	env = append(env, prefixEnv("ACCESS_TOKEN", cfg.AccessToken))
	env = append(env, prefixEnv("ADMIN_TOKEN", cfg.AdminToken))
	env = append(env, prefixEnv("AGENT", target))

	env = append(env, prefixEnv("VERBOSE", strconv.Itoa(cfg.Verbose)))
	env = append(env, prefixEnv("QUIET", strconv.FormatBool(cfg.Quiet)))

	env = append(env, prefixEnv("CONFIG_FILE", cfg.ConfigFile))

	env = append(env, prefixEnv("PROMPT_STATIC_PASSWORD", strconv.FormatBool(cfg.PromptPassword)))
	if cfg.GetStaticPassword() != "" {
		env = append(env, prefixEnv("PASSWORD", cfg.GetStaticPassword()))
	}

	return env
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
	Args []string `hidden:"true" arg:"" optional:"" passthrough:""`
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

type Tui struct {
	AuditMode bool `default:"false" name:"audit-mode" yaml:"audit-mode" help:"Redact all information in the TUI."`
}

// Run executes the tui subcommand.
func (t *Tui) Run(clientAPI *api.API, cfg ClientConfig) error {
	tt := tui.NewTui(clientAPI, cfg.AgentPassword, t.AuditMode)
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

		return cfg.SSH.Run(clientAPI, cfg)
	case "vscode":
		cfg.VsCode.Target = agent

		return cfg.VsCode.Run(clientAPI, cfg)
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
			if strings.HasPrefix(ctx.Command(), "compile") {
				_, _, _ = compiler.InitCompilerConfig(AppName, defaultValues)

				return nil
			}
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

	if cfg.Quiet {
		log.SetLogLevel(-1)
	} else {
		log.SetLogLevel(cfg.Verbose)
	}

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

func (cfg *ClientConfig) GetStaticPassword() string {
	if cfg.PrivatePassword != "" {
		return cfg.PrivatePassword
	}
	// If for instance a static password is set but the user currently wants to connect without a password,
	// an empty environment variable "TEALC_PASSWORD=" will be set
	// If we encounter it, we return an empty static password
	empty := prefixEnv("PASSWORD", "")
	for _, e := range os.Environ() {
		if e == empty {
			return ""
		}
	}
	pass, ok := cfg.AgentPassword[cfg.Pass.Agent]
	if ok {
		return pass
	}
	log.Debug().Str("Agent", cfg.Pass.Agent).Msg("No static password found, trying empty static password")

	return ""
}
