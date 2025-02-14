package config

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"Goauld/common"
	"Goauld/common/cli"
	"Goauld/common/log"
	"Goauld/common/ssh"
	"Goauld/common/utils"
	"github.com/alecthomas/kong"
)

var (
	_agePubKey = "age1fz7j9zck3qmafdkynu3ldvkjdrsstanhz8py8scx07hw7vja7aysuccrtn"

	_localSshPassword = ""
	_name             = "user@hostname"

	_sshd                   = "true"
	_socks                  = "true"
	_socks_use_system_proxy = "true"
	_proxy                  = ""
	_no_proxy               = "false"

	_server     = "localhost"
	_ssh_server = "localhost:2222"
	_tls_server = "localhost"

	// _sshd_port  = "0"
	_rssh_port  = "0"
	_socks_port = "0"

	_keepalive = "20"
	_verbosity = "0"

	_rssh_order = "SSH,TLS,WS,HTTP"

	_remote_port_forwarding = ""

	_max_retries = "0"

	_generate_config = "false"
	_config_file     = ""

	defaultValues = kong.Vars{
		"_age_pubkey": _agePubKey,

		"_localSshPassword": _localSshPassword,
		"_name":             _name,

		"_sshd":                   _sshd,
		"_socks":                  _socks,
		"_socks_use_system_proxy": _socks_use_system_proxy,
		"_proxy":                  _proxy,
		"_no_proxy":               _no_proxy,

		"_server":     _server,
		"_ssh_server": _ssh_server,
		"_tls_server": _tls_server,

		// "_sshd_port":  _sshd_port,
		"_rssh_port":  _rssh_port,
		"_socks_port": _socks_port,

		"_keepalive": _keepalive,
		"_verbosity": _verbosity,

		"_rssh_order": _rssh_order,

		"_remote_port_forwarding": _remote_port_forwarding,

		"_max_retries": _max_retries,

		"_generate_config": _generate_config,
		"_config_file":     _config_file,
	}
)

var (
	description = "Agent used to initiate the connection to the agent." +
		"\nThe agent will try to load configuration from " + filepath.Join("$HOME", ".config", strings.ToLower(common.AppName()), "agent_config.yaml") + "\n" +
		"\nAs well as agent_config.yaml on the current directory."
)

type AgentConfig struct {
	AgePubKey string `default:"${_age_pubkey}" help:"Age public key associated to the server. The provided public key should match the server public key" name:"age-pubkey" short:"A"`

	LocalSshPassword string `default:"${_localSshPassword}" short:"p" name:"password" optional:"" help:"SSH password to access the agent.\nIf not password is provided, an random password is automatically generated."`
	Name             string `default:"${_name}" name:"name" optional:"" help:"Nice name to identify the agent. Defaults to 'user@hostname'"`

	Sshd                bool     `default:"${_sshd}" name:"sshd" optional:"" negatable:"" help:"Start the SSHD server."`
	Socks               bool     `default:"${_socks}" name:"socks" optional:"" negatable:"" help:"Start the Socks server."`
	SocksUseSystemProxy bool     `default:"${_socks_use_system_proxy}" name:"socks-proxy" optional:"" negatable:"" help:"Use the proxy of the underlying system if applicable for all requests going through the socks proxy."`
	Proxy               *url.URL `default:"${_proxy}" name:"proxy" optional:"" help:"Use the provided proxy to connect the control server. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	NoProxy             bool     `default:"${_no_proxy}" name:"no-proxy" optional:"" help:"Don't use the system proxy."`

	Server    string `default:"${_server}" short:"s" name:"server" optional:"" help:"THe control HTTP server  to connect to."`
	SshServer string `default:"${_ssh_server}" short:"S" name:"ssh-server" optional:"" help:"The SSH server to connect to when using direct SSH connections."`
	TlsServer string `default:"${_tls_server}" short:"T" name:"tls-server" optional:"" help:"The TLS server to connect to when using SSH over TLS connections."`

	// SshdPort  int `default:"${_sshd_port}"  name:"sshd-port" optional:"" help:"Local port to listen to, 0 => Random."`
	RsshPort  int `default:"${_rssh_port}"  name:"rssh-port" optional:"" help:"The remote SSH port to bind to on the server.\n By default, the port is 0 meaning the port will be random on the server."`
	SocksPort int `default:"${_rssh_port}"  name:"socks-port" short:"D" optional:"" help:"The remote SOCKS port to bind to on the server,\n By default, the port is 0 meaning the port will be random on the server."`

	KeepAlive int `default:"${_keepalive}" short:"K"  name:"keepalive" optional:"" help:"Seconds between two keepalive messages in seconds, reduce this value if the connection drops."`
	Verbose   int `default:"${_verbosity}" help:"Verbosity of the logs. Repeat -v to increase" name:"verbose" short:"v" type:"counter"`

	RsshOrder []string `default:"${_rssh_order}" short:"O"  name:"rssh-order" optional:"" help:"Order the SSH tunnels connection attempts."`

	RemotePortForwarding []ssh.RemotePortForwarding `default:"${_remote_port_forwarding}" name:"rpf" short:"R" optional:"" help:"Ports to forward to the server (REMOTE_PORT[:LOCAL_IP]:LOCAL_PORT).\nIf REMOTE_PORT is 0, the port will be randomly chosen on the server"`

	MaxRetries int `default:"${_max_retries}" help:"Max retries connection attempts before giving up" name:"max-retries" short:"M"`

	GenerateConfig bool   `default:"${_generate_config}" help:"Generate configuration file based on the current options."`
	ConfigFile     string `name:"config-file" type:"existingfile" optionnal:"" short:"c" help:"Configuration file to use."`
}

// parse parses the command line arguments
func parse() (*kong.Context, *AgentConfig, error) {
	cfgTmp := &AgentConfig{}
	dir, err := utils.GetCurrentDirectory()
	if err != nil {
		return nil, cfgTmp, err
	}
	configSearchDir := []string{
		filepath.Join(dir, "agent_config.yaml"),
	}
	home, err := os.UserHomeDir()
	if err == nil {
		homeConfig := filepath.Join(home, ".config", strings.ToLower(common.AppName()), "agent_config.yaml")
		configSearchDir = append(configSearchDir, homeConfig)
	}
	kongOptions := []kong.Option{
		kong.Name(common.AppName()),
		kong.Description(description),
		kong.UsageOnError(),
		kong.Configuration(cli.YAMLKeepEnvVar, configSearchDir...),
		kong.DefaultEnvars(strings.ToUpper(common.AppName())),
		defaultValues,
	}
	_ = kong.Parse(cfgTmp, kongOptions...)

	if cfgTmp.ConfigFile != "" {
		kongOptions = append(kongOptions, kong.Configuration(cli.YAMLOverwriteEnvVar, cfgTmp.ConfigFile))
	}
	cfg := &AgentConfig{}
	app := kong.Parse(cfg, kongOptions...)

	log.SetLogLevel(cfg.Verbose)
	cfg.RsshOrder = utils.ToLower(utils.Unique(cfg.RsshOrder))
	return app, cfg, nil
}
