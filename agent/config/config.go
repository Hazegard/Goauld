package config

import (
	"Goauld/common"
	"Goauld/common/ssh"
	"net/url"
	"path/filepath"

	"github.com/alecthomas/kong"
)

var (
	_agePubKey = "age1e4txlmjtmc4sx5f8s7fhpka64d4d05rj3qn3jy4tgrta4p22euvq00ac5p"

	_disable_password = "false"         //nolint:revive
	_private_password = ""              //nolint:revive
	_password_cli     = ""              //nolint:revive
	_shared_password  = ""              //nolint:revive
	_name             = "user@hostname" //nolint:revive

	_server         = "www.example.com"
	_ssh_server     = "www.example.com:22222"      //nolint:revive
	_tls_server     = "app.example.com"            //nolint:revive
	_quic_domain    = "app.example.com"            //nolint:revive
	_dns_server     = "tns.example.com,8.8.8.8,1.1.1.1,9.9.9.9" //nolint:revive
	_dns_domain     = "t.example.com"                           //nolint:revive
	_dns_domain_alt = "s.example.com"                           //nolint:revive

	_sshd_enabled            = "true"  //nolint:revive
	_socks_enabled           = "true"  //nolint:revive
	_http_proxy_enabled      = "true"  //nolint:revive
	_mitm_http_proxy_enabled = "false" //nolint:revive
	_wg_enabled              = "false" //nolint:revive
	_relay_enabled           = "false" //nolint:revive
	_socks_upstream_proxy    = "http"  //nolint:revive

	_mitm_http_proxy_username = "" //nolint:revive
	_mitm_http_proxy_password = "" //nolint:revive
	_mitm_http_proxy_domain   = "" //nolint:revive

	_proxy          = ""
	_proxy_username = "" //nolint:revive
	_proxy_password = "" //nolint:revive
	_proxy_domain   = "" //nolint:revive

	_socks_custom_proxy   = "" //nolint:revive
	_socks_proxy_username = "" //nolint:revive
	_socks_proxy_password = "" //nolint:revive
	_socks_proxy_domain   = "" //nolint:revive

	_http_custom_proxy   = "" //nolint:revive
	_http_proxy_username = "" //nolint:revive
	_http_proxy_password = "" //nolint:revive
	_http_proxy_domain   = "" //nolint:revive

	_no_proxy = "false" //nolint:revive

	_rssh_port      = "0" //nolint:revive
	_socks_port     = "0" //nolint:revive
	_http_port      = "0" //nolint:revive
	_mitm_http_port = "0" //nolint:revive
	_wg_port        = "0" //nolint:revive
	_relay_port     = "0" //nolint:revive

	_keepawake = "false"
	_keepalive = "20"
	_verbosity = "0"
	_quiet     = "false"

	_only_working_days    = "false"        //nolint:revive
	_working_day_start    = "8:00"         //nolint:revive
	_working_day_end      = "19:30"        //nolint:revive
	_working_day_timezone = "Europe/Paris" //nolint:revive

	_rssh_order   = "SSH,TLS,WS,HTTP,DNS" //nolint:revive
	_rssh_timeout = "60"                  //nolint:revive

	_remote_port_forwarding = "" //nolint:revive

	_max_retries = "0" //nolint:revive

	_version         = "false"
	_generate_config = "false" //nolint:revive
	_config_file     = ""      //nolint:revive

	_background        = "false"
	_hidden_background = "false" //nolint:revive

	_custom_dns_command = "" //nolint:revive

	_relay_addr = ""

	_killswitch = "7"

	defaultValues = kong.Vars{
		"_agePubKey": _agePubKey,

		"_disable_password": _disable_password,
		"_password_cli":     _password_cli,
		"_shared_password":  _shared_password,
		"_name":             _name,

		"_sshd_enabled":            _sshd_enabled,
		"_socks_enabled":           _socks_enabled,
		"_http_proxy_enabled":      _http_proxy_enabled,
		"_mitm_http_proxy_enabled": _mitm_http_proxy_enabled,
		"_wg_enabled":              _wg_enabled,
		"_relay_enabled":           _relay_enabled,
		"_socks_upstream_proxy":    _socks_upstream_proxy,

		"_mitm_http_proxy_username": _mitm_http_proxy_username,
		"_mitm_http_proxy_password": _mitm_http_proxy_password,
		"_mitm_http_proxy_domain":   _mitm_http_proxy_domain,

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

		"_server":         _server,
		"_ssh_server":     _ssh_server,
		"_tls_server":     _tls_server,
		"_quic_domain":    _quic_domain,
		"_dns_server":     _dns_server,
		"_dns_domain":     _dns_domain,
		"_dns_domain_alt": _dns_domain_alt,

		"_rssh_port":      _rssh_port,
		"_socks_port":     _socks_port,
		"_http_port":      _http_port,
		"_mitm_http_port": _mitm_http_port,
		"_wg_port":        _wg_port,
		"_relay_port":     _relay_port,

		"_keepawake": _keepawake,
		"_keepalive": _keepalive,
		"_verbosity": _verbosity,
		"_quiet":     _quiet,

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

		"_background":        _background,
		"_hidden_background": _hidden_background,

		"_custom_dns_command": _custom_dns_command,

		"_killswitch": _killswitch,

		"_relay_addr": _relay_addr,

		"version": common.GetVersion(),
	}
)

var (
	description = "Agent used to initiate the connection to the agent." +
		"\nThe agent will try to load configuration from " + filepath.Join("$HOME", ".config", "goauld_agent.yaml") + "\n" +
		"\nAs well as goauld_agent.yaml on the current directory."
)

// AgentConfig the agent configuration.
type AgentConfig struct {
	AgePubKey string `default:"${_agePubKey}" group:"Agent configuration:" name:"age-pubkey" yaml:"age-pubkey" short:"A" help:"Age public key associated with the server. Must match the server’s public key."`

	Server             string   `default:"${_server}" group:"Control server configuration:" short:"s" name:"server" yaml:"server" optional:"" help:"Control HTTP server to connect to."`
	SSHServer          string   `default:"${_ssh_server}" group:"Control server configuration:" short:"S" name:"ssh-server" yaml:"ssh-server" optional:"" help:"SSH server used for direct SSH connections."`
	QuicServer         string   `default:"${_quic_domain}" group:"Control server configuration:" short:"Q" name:"quic-domain" yaml:"quic-domain" optional:"" help:"QUIC domain used to tunnel traffic."`
	TLSServer          string   `default:"${_tls_server}" group:"Control server configuration:" short:"T" name:"tls-server" yaml:"tls-server" optional:"" help:"TLS server used for SSH-over-TLS connections."`
	DNSServer          []string `default:"${_dns_server}" group:"Control server configuration:" short:"d" name:"dns-server" yaml:"dns-server" optional:"" help:"DNS servers used for SSH-over-DNS connections. Use 'system' to include the system DNS servers."`
	DNSServerDomain    string   `default:"${_dns_domain}" group:"Control server configuration:" short:"N" name:"dns-domain" yaml:"dns-domain" optional:"" help:"DNS domain used to tunnel SSH-over-DNS traffic."`
	DNSServerDomainAlt string   `default:"${_dns_domain_alt}" group:"Control server configuration:" name:"dns-domain-alt" yaml:"dns-domain-alt" optional:"" help:"DNS domain used to tunnel SSH-over-DNS traffic."`

	LocalSSHPassword string `default:"${_shared_password}" group:"Agent configuration:" short:"P" name:"shared-password" yaml:"shared-password" optional:"" hidden:"" help:"Password for local SSH access. If empty, a random password is generated automatically."`
	PrivatePassword  string `default:"${_password_cli}" group:"Agent configuration:" short:"p" name:"password" yaml:"password" optional:"" help:"Password required to access the agent."`
	DisablePassword  bool   `default:"${_disable_password}" group:"Agent configuration:" name:"disable-password" yaml:"disable-password" optional:"" help:"Disable the local password generation if no static password is provided."`
	Name             string `default:"${_name}" name:"name" group:"Agent configuration:" yaml:"name" optional:"" help:"Friendly name to identify the agent (default: 'user@hostname')."`

	Sshd     bool `default:"${_sshd_enabled}" group:"Agent features:" name:"sshd" yaml:"sshd" optional:"" negatable:"" help:"Enable the SSHD service."`
	Socks    bool `default:"${_socks_enabled}" group:"Agent features:" name:"socks" yaml:"socks" optional:"" negatable:"" help:"Enable the SOCKS proxy service."`
	HTTP     bool `default:"${_http_proxy_enabled}" group:"Agent features:" name:"http" yaml:"http" optional:"" negatable:"" help:"Enable the HTTP proxy service."`
	MITMHTTP bool `default:"${_mitm_http_proxy_enabled}" group:"Agent features:" name:"mitm-http" yaml:"mitm-http" optional:"" negatable:"" help:"Enable the MITM HTTP proxy service."`
	WG       bool `default:"${_wg_enabled}" group:"Agent features:" name:"wg" yaml:"wg" optional:"" negatable:"" help:"Enable the WireGuard service."`
	Relay    bool `default:"${_relay_enabled}" group:"Agent features:" name:"relay" yaml:"relay" optional:"" negatable:"" help:"Enable the Relay service."`

	MITMHTTPProxyUsername string `default:"${_mitm_http_proxy_username}" group:"HTTP MITM proxy custom configuration:" name:"mitm-http-proxy-username" yaml:"mitm-http-proxy-username" optional:"" help:"Username for the MITM HTTP upstream proxy."`
	MITMHTTPProxyPassword string `default:"${_mitm_http_proxy_password}" group:"HTTP MITM proxy custom configuration:" name:"mitm-http-proxy-password" yaml:"mitm-http-proxy-password" optional:"" help:"Password for the MITM HTTP upstream proxy."`
	MITMHTTPProxyDomain   string `default:"${_mitm_http_proxy_domain}" group:"HTTP MITM proxy custom configuration:" name:"mitm-http-proxy-domain" yaml:"mitm-http-proxy-domain" optional:"" help:"Domain for the MITM HTTP upstream proxy."`

	Proxy         *url.URL `default:"${_proxy}" group:"Egress proxy configuration:" name:"proxy" yaml:"proxy" optional:"" help:"Proxy URL to use for control server connections. If omitted, the system proxy is used (if configured)."`
	ProxyUsername string   `default:"${_proxy_username}" group:"Egress proxy configuration:" name:"proxy-username" yaml:"proxy-username" optional:"" help:"Username for the proxy server."`
	ProxyPassword string   `default:"${_proxy_password}" group:"Egress proxy configuration:" name:"proxy-password" yaml:"proxy-password" optional:"" help:"Password for the proxy server."`
	ProxyDomain   string   `default:"${_proxy_domain}" group:"Egress proxy configuration:" name:"proxy-domain" yaml:"proxy-domain" optional:"" help:"Authentication domain for the proxy server."`

	NoProxy bool `default:"${_no_proxy}" group:"Egress proxy configuration:" name:"no-proxy" yaml:"no-proxy" optional:"" help:"Ignore system proxy settings."`

	SocksCustomProxy   *url.URL `default:"${_socks_custom_proxy}" group:"Socks proxy custom configuration:" name:"socks-custom-proxy" yaml:"socks-custom-proxy" optional:"" help:"Custom proxy used within the SOCKS proxy. Falls back to the system proxy if not set."`
	SocksUpstreamProxy string   `default:"${_socks_upstream_proxy}" group:"Socks proxy custom configuration:" name:"socks-proxy" yaml:"socks-proxy" enum:"none,system,http,mitm" optional:"" help:"Configure the upstream HTTP proxy to use (none|system|http|mitm|custom)."`
	SocksProxyUsername string   `default:"${_socks_proxy_username}" group:"Socks proxy custom configuration:" name:"socks-proxy-username" yaml:"socks-proxy-username" optional:"" help:"Username for the SOCKS upstream proxy."`
	SocksProxyPassword string   `default:"${_socks_proxy_password}" group:"Socks proxy custom configuration:" name:"socks-proxy-password" yaml:"socks-proxy-password" optional:"" help:"Password for the SOCKS upstream proxy."`
	SocksProxyDomain   string   `default:"${_socks_proxy_domain}" group:"Socks proxy custom configuration:" name:"socks-proxy-domain" yaml:"socks-proxy-domain" optional:"" help:"Domain for the SOCKS upstream proxy."`

	HTTPCustomProxy   *url.URL `default:"${_http_custom_proxy}" group:"HTTP proxy custom configuration:" name:"http-custom-proxy" yaml:"http-custom-proxy" optional:"" help:"Custom proxy used within the HTTP proxy. Falls back to the system proxy if not set."`
	HTTPProxyUsername string   `default:"${_http_proxy_username}" group:"HTTP proxy custom configuration:" name:"http-proxy-username" yaml:"http-proxy-username" optional:"" help:"Username for the HTTP upstream proxy."`
	HTTPProxyPassword string   `default:"${_http_proxy_password}" group:"HTTP proxy custom configuration:" name:"http-proxy-password" yaml:"http-proxy-password" optional:"" help:"Password for the HTTP upstream proxy."`
	HTTPProxyDomain   string   `default:"${_http_proxy_domain}" group:"HTTP proxy custom configuration:" name:"http-proxy-domain" yaml:"http-proxy-domain" optional:"" help:"Domain for the HTTP upstream proxy."`

	RSSHPort          int `default:"${_rssh_port}" group:"Remote ports configuration:" name:"rssh-port" yaml:"rssh-port" optional:"" help:"Remote SSH port to bind on the server (0 = random)."`
	SocksPort         int `default:"${_socks_port}" group:"Remote ports configuration:" name:"socks-port" yaml:"socks-port" short:"D" optional:"" help:"Remote SOCKS proxy port to bind on the server (0 = random)."`
	HTTPProxyPort     int `default:"${_http_port}" group:"Remote ports configuration:" name:"http-port" yaml:"http-port" optional:"" help:"Remote HTTP proxy port to bind on the server (0 = random)."`
	MITMHTTPProxyPort int `default:"${_mitm_http_port}" group:"Remote ports configuration:" name:"mitm-http-port" yaml:"mitm-http-port" optional:"" help:"Remote MITM HTTP proxy port to bind on the server (0 = random)."`
	WGPort            int `default:"${_wg_port}" group:"Remote ports configuration:" name:"wg-port" yaml:"wg-port" optional:"" help:"Remote WireGuard port to bind on the server (0 = random)."`
	RelayPort         int `default:"${_relay_port}" group:"Remote ports configuration:" name:"relay-port" yaml:"relay-port" optional:"" help:"Remote WireGuard port to bind on the server (0 = random)."`

	KeepAwake bool `default:"${_keepawake}" group:"Agent configuration:" name:"keep-awake" yaml:"keep-awake" optional:"" help:"Prevent the system from sleeping or locking."`
	KeepAlive int  `default:"${_keepalive}" group:"Agent configuration:" short:"K" name:"keepalive" yaml:"keepalive" optional:"" help:"Interval in seconds between keepalive messages (0 = disabled)."`
	Verbose   int  `default:"${_verbosity}" group:"Agent configuration:" short:"v" name:"verbose" yaml:"verbose" short:"v" optional:"" type:"counter" help:"Increase log verbosity. Repeat for more detail."`
	Quiet     bool `default:"${_quiet}" group:"Agent configuration:" short:"q" name:"quiet" yaml:"quiet" short:"q" optional:"" help:"Suppress all log output."`

	OnlyWorkingDays    bool   `default:"${_only_working_days}" group:"Working days configuration:" name:"only-working-days" yaml:"only-working-days" optional:"" help:"Restrict agent activity to working days only."`
	WorkingDayStart    string `default:"${_working_day_start}" group:"Working days configuration:" name:"working-day-start" yaml:"working-day-start" optional:"" help:"Start time of the working day (e.g. '09:00')."`
	WorkingDayEnd      string `default:"${_working_day_end}" group:"Working days configuration:" name:"working-day-end" yaml:"working-day-end" optional:"" help:"End time of the working day (e.g. '17:00')."`
	WorkingDayTimeZone string `default:"${_working_day_timezone}" group:"Working days configuration:" name:"working-day-timezone" yaml:"working-day-timezone" optional:"" help:"Timezone used for working day calculations."`

	RSSHOrder  []string `default:"${_rssh_order}" group:"SSH configuration:" short:"O" name:"rssh-order" yaml:"rssh-order" optional:"" help:"Preferred order of SSH tunnel protocols."`
	SSHTimeout int      `default:"${_rssh_timeout}" group:"SSH configuration:"  name:"ssh-timeout" yaml:"ssh-timeout" optional:"" help:"Timeout in seconds to wait for each SSH tunnel attempt (0 = unlimited)."`

	RemotePortForwarding []ssh.RemotePortForwarding `default:"${_remote_port_forwarding}" group:"SSH configuration:" group:"SSH configuration"  short:"R" name:"rpf" yaml:"rpf" optional:""  help:"Ports to forward to the server (REMOTE_PORT[:LOCAL_IP]:LOCAL_PORT). Use 0 for a random remote port."`

	MaxRetries int `default:"${_max_retries}" group:"Agent configuration:" short:"M" name:"max-retries" yaml:"max-retries" help:"Maximum number of connection retries before giving up."`

	Version        bool   `default:"${_version}" group:"Agent configuration:"  short:"V" name:"version" yaml:"version" help:"Show version information and exit."`
	GenerateConfig bool   `default:"${_generate_config}" group:"Agent configuration:"  name:"generate-config" yaml:"generate-config" help:"Generate a configuration file from the current settings."`
	ConfigFile     string `default:"${_config_file}" group:"Agent configuration:"  short:"c" name:"config-file" yaml:"config-file" optional:"" help:"Path to the configuration file to use."`

	Background       bool `default:"${_background}" group:"Agent configuration:"  short:"B" name:"background" yaml:"background" negatable:"" optional:"" help:"Run the agent in the background."`
	HiddenBackground bool `default:"${_hidden_background}" name:"hidden-background" yaml:"hidden-background" hidden:"" negatable:"" optional:"" help:"Run the agent in hidden background mode."`

	CustomDNSCommand string `default:"${_custom_dns_command}" group:"Agent configuration:"  name:"custom-dns-command" yaml:"custom-dns-command" help:"System command used to perform SSH over DNS when raw DNS queries are blocked. The provided command is responsible for performing the DNS query and returning the result as raw bytes.\n Powershell example: \"((Resolve-DnsName -Type TXT -Server 127.0.0.1 '%s')[0].Strings -join '\x00' -replace '\\s+', '\x00' -split '..' | ForEach-Object { [Convert]::ToByte($_,16) } )\"\n Linux example:\"dig +short +unknownformat -t TXT '%s' @127.0.0.1 | head -n1 | cut -d ' ' -f3- | tr -d ' '  | xxd -r -p\"."`

	KillSwitch int `default:"${_killswitch}" group:"Agent configuration:"  name:"kill-switch" yaml:"kill-switch" help:"Number of days before the agent self-terminates (0 = disabled)."`

	RelayAddr string `default:"${_relay_addr}" group:"Agent configuration:"  name:"relay-addr" yaml:"relay-addr" help:"Use another agent to relay the connection to the server."`

	Remaining []string `arg:"" name:"remaining" yaml:"remaining" passthrough:"" optional:"" hidden:"" help:"Extra arguments that will be trashed, required when launching the agent in some contexts."`
}
