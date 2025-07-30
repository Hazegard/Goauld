package config

import (
	"errors"
	"fmt"
	"golang.org/x/crypto/bcrypt"
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
	_agePubKey = "age1e4txlmjtmc4sx5f8s7fhpka64d4d05rj3qn3jy4tgrta4p22euvq00ac5p"

	_private_password     = ""
	_private_password_cli = ""
	_shared_password      = ""
	_name                 = "user@hostname"

	_server      = "www.example.com"
	_ssh_server  = "www.example.com:22222"
	_tls_server  = "app.example.com"
	_quic_domain = "app.example.com"
	_dns_server  = "tns.example.com,8.8.8.8,1.1.1.1,9.9.9.9"
	_dns_domain  = "t.example.com"

	_sshd_enabled           = "true"
	_socks_enabled          = "true"
	_http_proxy_enabled     = "true"
	_socks_use_system_proxy = "true"

	_proxy          = ""
	_proxy_username = ""
	_proxy_password = ""
	_proxy_domain   = ""

	_socks_custom_proxy   = ""
	_socks_proxy_username = ""
	_socks_proxy_password = ""
	_socks_proxy_domain   = ""

	_http_custom_proxy   = ""
	_http_proxy_username = ""
	_http_proxy_password = ""
	_http_proxy_domain   = ""

	_no_proxy = "false"

	// _sshd_port  = "0"
	_rssh_port  = "0"
	_socks_port = "0"
	_http_port  = "0"

	_keepawake = "false"
	_keepalive = "20"
	_verbosity = "0"

	_only_working_days    = "false"
	_working_day_start    = "8:00"
	_working_day_end      = "19:30"
	_working_day_timezone = "Europe/Paris"

	_rssh_order   = "SSH,TLS,WS,HTTP,DNS"
	_rssh_timeout = "60"

	_remote_port_forwarding = ""

	_max_retries = "0"

	_version         = "false"
	_generate_config = "false"
	_config_file     = ""

	_background = "false"

	defaultValues = kong.Vars{
		"_agePubKey": _agePubKey,

		"_private_password_cli": _private_password_cli,
		"_shared_password":      _shared_password,
		"_name":                 _name,

		"_sshd_enabled":           _sshd_enabled,
		"_socks_enabled":          _socks_enabled,
		"_http_proxy_enabled":     _http_proxy_enabled,
		"_socks_use_system_proxy": _socks_use_system_proxy,

		"_no_proxy":       _no_proxy,
		"_proxy":          _proxy,
		"_proxy_username": _proxy_username,
		"_proxy_password": _proxy_password,
		"_proxy_domain":   _proxy_domain,

		"_socks_custom_proxy":   _socks_custom_proxy,
		"_socks_proxy_username": _socks_proxy_username,
		"_socks_proxy_password": _socks_proxy_password,
		"_socks_proxy_domain":   _socks_proxy_domain,

		"_http_custom_proxy":   _http_custom_proxy,
		"_http_proxy_username": _http_proxy_username,
		"_http_proxy_password": _http_proxy_password,
		"_http_proxy_domain":   _http_proxy_domain,

		"_server":      _server,
		"_ssh_server":  _ssh_server,
		"_tls_server":  _tls_server,
		"_quic_domain": _quic_domain,
		"_dns_server":  _dns_server,
		"_dns_domain":  _dns_domain,

		"_rssh_port":  _rssh_port,
		"_socks_port": _socks_port,
		"_http_port":  _http_port,

		"_keepawake": _keepawake,
		"_keepalive": _keepalive,
		"_verbosity": _verbosity,

		"_only_working_days":    _only_working_days,
		"_working_day_start":    _working_day_start,
		"_working_day_end":      _working_day_end,
		"_working_day_timezone": _working_day_timezone,

		"_rssh_order":   _rssh_order,
		"_rssh_timeout": _rssh_timeout,

		"_remote_port_forwarding": _remote_port_forwarding,

		"_max_retries": _max_retries,

		"_version":         _version,
		"_generate_config": _generate_config,
		"_config_file":     _config_file,

		"_background": _background,
	}
)

var (
	description = "Agent used to initiate the connection to the agent." +
		"\nThe agent will try to load configuration from " + filepath.Join("$HOME", ".config", "goauld_agent.yaml") + "\n" +
		"\nAs well as goauld_agent.yaml on the current directory."
)

type AgentConfig struct {
	AgePubKey string `default:"${_agePubKey}" name:"age-pubkey" yaml:"age-pubkey" short:"A" help:"Age public key associated to the server. The provided public key should match the server public key"`

	Server          string   `default:"${_server}" short:"s" name:"server" yaml:"server" optional:"" help:"The control HTTP server to connect to."`
	SshServer       string   `default:"${_ssh_server}" short:"S" name:"ssh-server" yaml:"ssh-server" optional:"" help:"The SSH server to connect to when using direct SSH connections."`
	QuicServer      string   `default:"${_quic_domain}" short:"Q" name:"quic-domain" yaml:"quic-domain" optional:"" help:"The QUIC domain used to tunnel the traffic."`
	TlsServer       string   `default:"${_tls_server}" short:"T" name:"tls-server" yaml:"tls-server" optional:"" help:"The TLS server to connect to when using SSH over TLS connections."`
	DnsServer       []string `default:"${_dns_server}" short:"d" name:"dns-server" yaml:"dns-server" optional:"" help:"The DNS server to connect to when using SSH over DNS connections, the magic name 'system' will be replaced by the list of the system DNS servers."`
	DnsServerDomain string   `default:"${_dns_domain}" short:"N" name:"dns-domain" yaml:"dns-domain" optional:"" help:"The DNS domain used to tunnel the traffic."`

	LocalSshPassword string `default:"${_shared_password}" short:"P" name:"shared-password" yaml:"shared-password" optional:"" hidden:"" help:"SSH password to access the agent. If no password is provided, a random password is automatically generated."`
	PrivatePassword  string `default:"${_private_password_cli}" short:"p" name:"password" optional:"" help:"SSH password to access the agent"`
	Name             string `default:"${_name}" name:"name" yaml:"name" optional:"" help:"Nice name to identify the agent. Defaults to 'user@hostname'"`

	Sshd  bool `default:"${_sshd_enabled}" name:"sshd" yaml:"sshd" optional:"" negatable:"" help:"Start the SSHD server."`
	Socks bool `default:"${_socks_enabled}" name:"socks" yaml:"socks" optional:"" negatable:"" help:"Start the Socks proxy server."`
	Http  bool `default:"${_http_proxy_enabled}" name:"http" yaml:"http" optional:"" negatable:"" help:"Start the Http proxy server."`

	Proxy         *url.URL `default:"${_proxy}" name:"proxy" yaml:"proxy" optional:"" help:"Use the provided proxy to connect the control server. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	ProxyUsername string   `default:"${_proxy_username}" name:"proxy-username" yaml:"proxy-username" optional:"" help:"Username to use with the proxy"`
	ProxyPassword string   `default:"${_proxy_password}" name:"proxy-password" yaml:"proxy-password" optional:"" help:"Password to use with the proxy"`
	ProxyDomain   string   `default:"${_proxy_domain}" name:"proxy-domain" yaml:"proxy-domain" optional:"" help:"Domain to use with the proxy"`

	SocksCustomProxy    *url.URL `default:"${_socks_custom_proxy}" name:"socks-custom-proxy" yaml:"socks-custom-proxy" optional:"" help:"Use the provided proxy to use within the socks proxy. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	SocksUseSystemProxy bool     `default:"${_socks_use_system_proxy}" name:"socks-proxy" yaml:"socks-proxy" optional:"" negatable:"" help:"Use the proxy of the underlying system if applicable for all requests going through the socks proxy."`
	SocksProxyUsername  string   `default:"${_socks_proxy_username}" name:"socks-proxy-username" yaml:"socks-proxy-username" optional:"" help:"Username to use with the socks http upstream proxy"`
	SocksProxyPassword  string   `default:"${_socks_proxy_password}" name:"socks-proxy-password" yaml:"socks-proxy-password" optional:"" help:"Password to use with the socks http upstream proxy"`
	SocksProxyDomain    string   `default:"${_socks_proxy_domain}" name:"socks-proxy-domain" yaml:"socks-proxy-domain" optional:"" help:"Domain to use with the socks http upstream proxy"`

	HttpCustomProxy   *url.URL `default:"${_http_custom_proxy}" name:"http-custom-proxy" yaml:"http-custom-proxy" optional:"" help:"Use the provided proxy to use within the http proxy. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	HttpProxyUsername string   `default:"${_http_proxy_username}" name:"http-proxy-username" yaml:"http-proxy-username" optional:"" help:"Username to use with the http upstream proxy"`
	HttpProxyPassword string   `default:"${_http_proxy_password}" name:"http-proxy-password" yaml:"http-proxy-password" optional:"" help:"Password to use with the http upstream proxy"`
	HttpProxyDomain   string   `default:"${_http_proxy_domain}" name:"http-proxy-domain" yaml:"http-proxy-domain" optional:"" help:"Domain to use with the http upstream proxy"`

	NoProxy bool `default:"${_no_proxy}" name:"no-proxy" yaml:"no-proxy" optional:"" help:"Do not use the system proxy."`

	RsshPort      int `default:"${_rssh_port}"  name:"rssh-port" yaml:"rssh-port" optional:"" help:"The remote SSH port to bind to on the server.  By default, the port is 0 meaning the port will be random on the server."`
	SocksPort     int `default:"${_socks_port}"  name:"socks-port" yaml:"socks-port" short:"D" optional:"" help:"The remote SOCKS proxy port to bind to on the server,  By default, the port is 0 meaning the port will be random on the server."`
	HttpProxyPort int `default:"${_http_port}"  name:"http-port" yaml:"http-port" short:"" optional:"" help:"The remote HTTP proxy port to bind to on the server,  By default, the port is 0 meaning the port will be random on the server."`

	KeepAwake bool `default:"${_keepawake}" name:"keep-awake" yaml:"keep-awake" optional:"" help:"Keep the system awake (try to prevent from sleep and lock screen)."`
	KeepAlive int  `default:"${_keepalive}" short:"K"  name:"keepalive" yaml:"keepalive" optional:"" help:"Seconds between two keepalive messages in seconds, reduce this value if the connection drops (0 => no keepalive)."`
	Verbose   int  `default:"${_verbosity}" name:"verbose" yaml:"verbose" short:"v" type:"counter" help:"Verbosity of the logs. Repeat -v to increase"`

	OnlyWorkingDays    bool   `default:"${_only_working_days}" name:"only-working-days" yaml:"only-working-days" optional:"" help:"Only working days."`
	WorkingDayStart    string `default:"${_working_day_start}" name:"working-day-start" yaml:"working-day-start" help:"Start time of working day in days."`
	WorkingDayEnd      string `default:"${_working_day_end}" name:"working-day-end" yaml:"working-day-end" help:"End time of working day in days."`
	WorkingDayTimeZone string `default:"${_working_day_timezone}" name:"working-day-timezone" yaml:"working-day-timezone" help:"Timezone of working day."`

	RsshOrder  []string `default:"${_rssh_order}" short:"O" name:"rssh-order" yaml:"rssh-order" optional:"" help:"Order the SSH tunnels connection attempts."`
	SshTimeout int      `default:"${_rssh_timeout}" name:"ssh-timeout" yaml:"ssh-timeout" help:"Timeout in second to wait for the SSH tunnel to become available (independent for each protocol attempt), 0 means wait indefinitely."`

	RemotePortForwarding []ssh.RemotePortForwarding `default:"${_remote_port_forwarding}" name:"rpf" yaml:"rpf"  short:"R" optional:"" help:"Ports to forward to the server (REMOTE_PORT[:LOCAL_IP]:LOCAL_PORT). If REMOTE_PORT is 0, the port will be randomly chosen on the server"`

	MaxRetries int `default:"${_max_retries}" name:"max-retries" yaml:"max-retries" short:"M" help:"Max retries connection attempts before giving up"`

	Version        bool   `default:"${_version}" name:"version" yaml:"version" short:"V" help:"Show version information"`
	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" yaml:"generate-config" help:"Generate configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" name:"config-file" yaml:"config-file" optionnal:"" short:"c" help:"Configuration file to use."`

	Background       bool `name:"background" yaml:"background" short:"B" default:"${_background}" negatable:"" optional:"" help:"Start the agent in the background."`
	HiddenBackground bool `name:"hidden-background" yaml:"hidden-background" hidden:""  negatable:"" optional:"" help:"Start the agent in the background."`
}

func (c *AgentConfig) Validate() error {
	var errs []error
	if c.OnlyWorkingDays {
		wd := NewWorkingDay(c.WorkingDayStart, c.WorkingDayEnd, c.WorkingDayTimeZone)
		errs = append(errs, wd.Validate())
	}
	if HasProto(c.TlsServer) {
		errs = append(errs, fmt.Errorf("the TLS server name must not contains protocol prefix"))
	}
	if HasProto(c.QuicServer) {
		errs = append(errs, fmt.Errorf("the QUIC server name must not contains protocol prefix"))
	}
	if len(c.PrivatePassword) > 72 {
		errs = append(errs, bcrypt.ErrPasswordTooLong)
	}
	return errors.Join(errs...)
}

// parse parses the command line arguments
func parse() (*kong.Context, *AgentConfig, error) {
	cfgTmp := &AgentConfig{}
	dir, err := utils.GetCurrentDirectory()
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
		kong.Description(common.Title(common.App_Name) + "\n" + description),
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

	log.SetLogLevel(cfg.Verbose)
	return app, cfg, nil
}

// HasProto returns true if the url contains a protocol prefix
func HasProto(u string) bool {
	split := strings.Split(u, "://")
	return len(split) > 1
}
