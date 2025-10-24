// Package config holds the agent configuration
package config

import (
	"Goauld/common"
	"Goauld/common/crypto/pwgen"
	"Goauld/common/log"
	"Goauld/common/utils"

	//nolint:gosec
	"crypto/md5"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"strings"
	"time"

	"github.com/qdm12/dns/v2/pkg/nameserver"

	"Goauld/common/cli"

	"Goauld/common/crypto"
	"Goauld/common/ssh"

	"github.com/alecthomas/kong"
	"github.com/keygen-sh/machineid"
)

// Agent the dynamic agent configuration.
type Agent struct {
	ID                       string
	SSHPrivateKey            string
	SharedSecret             string
	Cryptor                  *crypto.SymCryptor
	cfg                      *AgentConfig
	IsStaticPasswordDynamic  bool
	RemoteDynamicPortForward []int
	RemotePortForward        []int
	Platform                 string
	Architecture             string
	Username                 string
	Hostname                 string
	IPs                      []string
	Path                     string
	WorkingDay               WorkingDay
}

var agent *Agent

// InitAgent parses the command lines arguments and initializes the temporary values (shared secret,etc...)
func InitAgent() (*kong.Context, []error, error) {
	var warnings []error
	// Parse the command line arguments
	ctx, cfg, err := parse()
	if err != nil {
		return nil, nil, fmt.Errorf("parsing arguments: %w", err)
	}
	// Generate the shared secret
	if cfg.GenerateConfig {
		agent = &Agent{
			cfg: cfg,
		}

		return ctx, warnings, nil
	}
	if cfg.AgePubKey == "" {
		return nil, nil, errors.New("AgePubKey is required")
	}
	sharedSecret, err := crypto.GeneratePassword(crypto.PasswordLength)
	if err != nil {
		return nil, nil, fmt.Errorf("error generating ssh password: %w", err)
	}
	// Generate the local password if not provided
	if cfg.LocalSSHPassword == "" {
		sshPassword, err := crypto.GeneratePassword(crypto.PasswordLength)
		if err != nil {
			return nil, nil, fmt.Errorf("error generating ssh password: %w", err)
		}
		cfg.LocalSSHPassword = sshPassword
	}

	// Generate the encryption mechanism
	cryptor, err := crypto.NewCryptor(sharedSecret)
	if err != nil {
		return nil, nil, fmt.Errorf("initializing cryptor: %w", err)
	}

	// Generate the agent name if not provided
	if cfg.Name == "user@hostname" {
		var name string
		userName, err := user.Current()
		if err != nil {
			return nil, nil, fmt.Errorf("error getting current user: %w", err)
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
			return nil, nil, fmt.Errorf("error getting hostname: %w", err)
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
		warnings = append(warnings, fmt.Errorf("error getting hostname: %w", err))
	}

	u, err := user.Current()
	if err != nil {
		warnings = append(warnings, fmt.Errorf("error getting user: %w", err))
	}
	ips, errs := getIPs()
	if len(errs) > 0 {
		warnings = append(warnings, fmt.Errorf("error getting ips: %w", errors.Join(errs...)))
	}

	currDir, err := os.Getwd()
	if err != nil {
		warnings = append(warnings, fmt.Errorf("error getting current directory: %w", err))
	}
	agent = &Agent{
		ID:                       id,
		SharedSecret:             sharedSecret,
		Cryptor:                  cryptor,
		cfg:                      cfg,
		RemoteDynamicPortForward: nil,
		RemotePortForward:        nil,
		Platform:                 runtime.GOOS,
		Architecture:             runtime.GOARCH,
		Username:                 u.Username,
		Hostname:                 host,
		IPs:                      ips,
		Path:                     currDir,
		WorkingDay:               *NewWorkingDay(cfg.WorkingDayStart, cfg.WorkingDayEnd, cfg.WorkingDayTimeZone),
	}

	return ctx, warnings, nil
}

// Get returns the Agent global object.
func Get() *Agent {
	return agent
}

// Verbosity returns the current log verbosity.
func (a *Agent) Verbosity() int {
	return a.cfg.Verbose
}

// LocalSSHDPassword returns the current ssh password allowing to connect.
func (a *Agent) LocalSSHDPassword() string {
	return a.cfg.LocalSSHPassword
}

// PrivateSshdPassword return the static password.
func (a *Agent) PrivateSshdPassword() string {
	if a.cfg.PrivatePassword != "" {
		return a.cfg.PrivatePassword
	}
	if _private_password == "" {
		p, err := pwgen.GetXKCDPassword()
		if err != nil {
			p, err = crypto.GeneratePassword(30)
			if err != nil {
				panic(err)
			}
		}
		a.IsStaticPasswordDynamic = true
		_private_password = p
	}
	return _private_password
}

// ValidatePassword return whether the incoming password is valid.
// The password is concatenated using the dynamic password, and the compiled-time-defined password
// (private sshd password).
func (a *Agent) ValidatePassword(in string) bool {
	return in == a.PrivateSshdPassword()+a.LocalSSHDPassword()
}

// DNSServer returns the DNS servers that will be used to tunnel the connection.
func (a *Agent) DNSServer() []string {
	var servers []string
	for _, srv := range a.cfg.DNSServer {
		if strings.EqualFold(srv, "system") {
			for _, systemDNSServer := range nameserver.GetDNSServers() {
				servers = append(servers, systemDNSServer.String())
			}
		} else {
			servers = append(servers, srv)
		}
	}

	return servers
}

// DNSDomain returns the DNS domain that will be used to query the DNS server.
func (a *Agent) DNSDomain() string {
	return a.cfg.DNSServerDomain
}

// ControlSSHServer returns the SSHD server.
func (a *Agent) ControlSSHServer() string {
	return a.cfg.SSHServer
}

// IsRemoteForwardedSshdPortRandom returns whether the remote forwarded SSHD port is random.
func (a *Agent) IsRemoteForwardedSshdPortRandom() bool {
	return a.cfg.RSSHPort == 0
}

// RsshPort returns the remote forwarded SSHD port.
func (a *Agent) RsshPort() int {
	return a.cfg.RSSHPort
}

// RemoteForwardedSshdAddress returns the remote forwarded SSHD address.
func (a *Agent) RemoteForwardedSshdAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.RSSHPort)
}

// RemoteForwardedSocksAddress returns the remote forwarded Socks address.
func (a *Agent) RemoteForwardedSocksAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.SocksPort)
}

// RemoteForwardedHTTPProxyAddress returns the remote forwarded Socks address.
func (a *Agent) RemoteForwardedHTTPProxyAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.HTTPProxyPort)
}

// RemoteForwardedHTTPProxyPort returns the remote forwarded HTTP port.
func (a *Agent) RemoteForwardedHTTPProxyPort() int {
	return a.cfg.HTTPProxyPort
}

// RemoteForwardedSocksPort returns the remote forwarded Socks port.
func (a *Agent) RemoteForwardedSocksPort() int {
	return a.cfg.SocksPort
}

// RemoteForwardedSshdPort returns the remote forwarded SSHD port.
func (a *Agent) RemoteForwardedSshdPort() int {
	return a.cfg.RSSHPort
}

// UpdateSocksPort set the new socks port.
func (a *Agent) UpdateSocksPort(port int) {
	a.cfg.SocksPort = port
}

// UpdateHTTPProxyPort set the new socks port.
func (a *Agent) UpdateHTTPProxyPort(port int) {
	a.cfg.HTTPProxyPort = port
}

// UpdateSshdPort set the new socks port.
func (a *Agent) UpdateSshdPort(port int) {
	a.cfg.RSSHPort = port
}

// ServerURL return the HTTP control server URL.
func (a *Agent) ServerURL() string {
	u := normalizeAddr(a.cfg.Server)
	if u != "" {
		return u
	}
	switch {
	case strings.HasPrefix(a.cfg.Server, "http://"):
		u = a.cfg.Server
	case strings.HasPrefix(a.cfg.Server, "https://"):
		u = a.cfg.Server
	default:
		u = "http://" + a.cfg.Server
	}

	return u
}

// normalizeAddr transform the incoming url to ensure that a scheme and a port are always present
// It tries to allocate the corresponding port/scheme (HTTPS/443, HTTP/80, etc.)
func normalizeAddr(input string) string {
	// 1) Split off any “scheme://”
	hasScheme := strings.Contains(input, "://")
	working := input
	if !hasScheme {
		// Tentatively prepend a fake scheme, so url.Parse will let us split Host vs Path
		working = "http://" + working
	}

	u, err := url.Parse(working)
	if err != nil {
		return ""
	}

	host := u.Hostname()
	port := u.Port()

	// 2) Decide on a scheme if missing
	scheme := u.Scheme
	if !hasScheme {
		if port == "443" {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	// 3) Decide on port if missing
	if port == "" {
		switch scheme {
		case "https":
			port = "443"
		default:
			port = "80"
		}
	}

	// Recombine
	// note: u.Path, u.User, u.RawQuery etc. are ignored here; this is just host:port normalization
	return fmt.Sprintf("%s://%s:%s", scheme, host, port)
}

// WSshURL return the SSH over Websocket connection URL.
func (a *Agent) WSshURL() string {
	return fmt.Sprintf("%s/wssh/%s", a.ServerURL(), a.ID)
}

// SocketIoURL return the control connection URL.
func (a *Agent) SocketIoURL() string {
	return fmt.Sprintf("%s/live/%s/", a.ServerURL(), a.ID)
}

// SSHTTPURL return the SSH over HTTP connection URL.
func (a *Agent) SSHTTPURL() string {
	return fmt.Sprintf("%s/sshttp/%s", a.ServerURL(), a.ID)
}

// TLSURL return the SSH over TLS connection URL.
func (a *Agent) TLSURL() string {
	u := strings.Split(a.cfg.TLSServer, ":")
	if len(u) == 1 {
		return a.cfg.TLSServer + ":443"
	}

	return a.cfg.TLSServer
}

// QuicURL return the SSH over TLS connection URL.
func (a *Agent) QuicURL() string {
	u := strings.Split(a.cfg.QuicServer, ":")
	if len(u) == 1 {
		return a.cfg.QuicServer + ":443"
	}

	return a.cfg.TLSServer
}

// Name returns the agent name.
func (a *Agent) Name() string {
	return a.cfg.Name
}

// KeepAwake returns whether the agent should try to keep the underlying system awake.
func (a *Agent) KeepAwake() bool {
	return a.cfg.KeepAwake
}

// GetKeepalive returns the duration between two keepalive pings.
func (a *Agent) GetKeepalive() time.Duration {
	return time.Duration(a.cfg.KeepAlive)
}

// OnlyWorkingDays return whether the OnlyWorkingDay option is enabled.
func (a *Agent) OnlyWorkingDays() bool {
	return a.cfg.OnlyWorkingDays
}

// IsOutOfWorkingDay returns whether the agent is configured to run only on working days
// AND if the current time is out of working day.
func (a *Agent) IsOutOfWorkingDay() bool {
	if !a.cfg.OnlyWorkingDays {
		return false
	}

	return a.cfg.OnlyWorkingDays && !a.WorkingDay.IsWorkingPeriod()
}

// NextStart return the next date when the agent will be allowed to start.
func (a *Agent) NextStart() (time.Time, time.Time, error) {
	return a.WorkingDay.NextStartAndNow()
}

// StartTime returns the start working day.
func (a *Agent) StartTime() string {
	return a.WorkingDay.Start
}

func (a *Agent) SetRSSHOrder(order []string) {
	a.cfg.RSSHOrder = order
}

// GetRSSHOrder returns the order that the agent should follow to attempt to connect
// to the SSHD server.
func (a *Agent) GetRSSHOrder() []string {
	return utils.ToLower(utils.Unique(a.cfg.RSSHOrder))
}

// GetSSHTimeout return the SSH timeout used when setting up the ssh connection.
func (a *Agent) GetSSHTimeout() time.Duration {
	if a.cfg.SSHTimeout == 0 {
		return time.Duration(1<<63 - 1)
	}

	return time.Duration(a.cfg.SSHTimeout) * time.Second
}

// AgePubKey returns the age public key used to asymmetrically encrypt data.
func (a *Agent) AgePubKey() string {
	return a.cfg.AgePubKey
}

// SshdEnabled returns whether the sshd server is enabled.
func (a *Agent) SshdEnabled() bool {
	return a.cfg.Sshd
}

// HTTPProxyEnabled returns whether the http proxy server is enabled.
func (a *Agent) HTTPProxyEnabled() bool {
	return a.cfg.HTTP
}

// SocksEnabled returns whether the socks server is enabled.
func (a *Agent) SocksEnabled() bool {
	return a.cfg.Socks || a.cfg.SocksPort != 0
}

// SocksUseSystemProxy returns whether the agent Socks proxy should use the system proxy (if applicable).
func (a *Agent) SocksUseSystemProxy() bool {
	return a.cfg.SocksUseSystemProxy
}

// AddSshdToRpf adds the SSHD conf to the Remote port forwarding list.
func (a *Agent) AddSshdToRpf() {
	sshdRpf := ssh.RemotePortForwarding{
		ServerPort: a.cfg.RSSHPort,
		AgentPort:  -1,
		AgentIP:    "0.0.0.0",
		Tag:        "sshd",
	}
	a.cfg.RemotePortForwarding = append(a.cfg.RemotePortForwarding, sshdRpf)
}

// NoProxy return whether the agent should ignore the potential system proxy.
func (a *Agent) NoProxy() bool {
	return a.cfg.NoProxy
}

// Proxy returns the proxy provided by the configuration.
func (a *Agent) Proxy() *url.URL {
	return a.cfg.Proxy
}

// ProxyUsername returns the username used by the proxy.
func (a *Agent) ProxyUsername() string {
	return a.cfg.ProxyUsername
}

// ProxyPassword returns the password used by the proxy.
func (a *Agent) ProxyPassword() string {
	return a.cfg.ProxyPassword
}

// ProxyDomain returns the domain used by the proxy.
func (a *Agent) ProxyDomain() string {
	return a.cfg.ProxyPassword
}

// SocksProxy returns the proxy provided by the configuration.
func (a *Agent) SocksProxy() *url.URL {
	if a.cfg.SocksCustomProxy != nil {
		return a.cfg.SocksCustomProxy
	}

	return a.cfg.Proxy
}

// SocksProxyUsername returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) SocksProxyUsername() string {
	if a.cfg.SocksProxyUsername != "" {
		return a.cfg.SocksProxyUsername
	}

	return a.cfg.ProxyUsername
}

// SocksProxyPassword returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) SocksProxyPassword() string {
	if a.cfg.SocksProxyPassword != "" {
		return a.cfg.SocksProxyPassword
	}

	return a.cfg.ProxyPassword
}

// SocksProxyDomain returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) SocksProxyDomain() string {
	if a.cfg.SocksProxyDomain != "" {
		return a.cfg.SocksProxyDomain
	}

	return a.cfg.ProxyDomain
}

// HTTPProxy returns the proxy provided by the configuration.
func (a *Agent) HTTPProxy() *url.URL {
	if a.cfg.SocksCustomProxy != nil {
		return a.cfg.SocksCustomProxy
	}

	return a.cfg.Proxy
}

// HTTPProxyUsername returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) HTTPProxyUsername() string {
	if a.cfg.SocksProxyUsername != "" {
		return a.cfg.SocksProxyUsername
	}

	return a.cfg.ProxyUsername
}

// HTTPProxyPassword returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) HTTPProxyPassword() string {
	if a.cfg.SocksProxyPassword != "" {
		return a.cfg.SocksProxyPassword
	}

	return a.cfg.ProxyPassword
}

// HTTPProxyDomain returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) HTTPProxyDomain() string {
	if a.cfg.SocksProxyDomain != "" {
		return a.cfg.SocksProxyDomain
	}

	return a.cfg.ProxyDomain
}

// AddSocksToRpf adds the SSHD conf to the Remote port forwarding list.
func (a *Agent) AddSocksToRpf() {
	sshdRpf := ssh.RemotePortForwarding{
		ServerPort: a.cfg.SocksPort,
		AgentPort:  -1,
		AgentIP:    "0.0.0.0",
		Tag:        "socks",
	}
	a.cfg.RemotePortForwarding = append(a.cfg.RemotePortForwarding, sshdRpf)
}

// GetRemotePortForwarding returns the configured remote port forwarding.
func (a *Agent) GetRemotePortForwarding() []ssh.RemotePortForwarding {
	return a.cfg.RemotePortForwarding
}

// getIPs returns the IP on the hosts, excluding local network addresses.
func getIPs() ([]string, []error) {
	IPS := make([]string, 0)

	errs := make([]error, 0)
	ifaces, err := net.Interfaces()
	if err != nil {
		errs = append(errs, err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			errs = append(errs, err)
		}
		for _, addr := range addrs {
			if strings.HasPrefix(addr.String(), "fe80:") || strings.HasPrefix(addr.String(), "127.") {
				continue
			}
			if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback == 0 {
				IPS = append(IPS, addr.String())
			}
		}
	}

	return IPS, errs
}

// GetMexRetries returns the configured max retries before killing the agent.
func (a *Agent) GetMexRetries() uint {
	//nolint:gosec
	return uint(a.cfg.MaxRetries)
}

// DoPrintVersion return whether the version must be printed.
func (a *Agent) DoPrintVersion() bool {
	return a.cfg.Version
}

// DoGenerateConfig return whether the configuration generation should be enabled.
func (a *Agent) DoGenerateConfig() bool {
	return a.cfg.GenerateConfig
}

// ShouldRunInBackground returns whether the agent should be run in the background.
func (a *Agent) ShouldRunInBackground() bool {
	return a.cfg.Background && !a.cfg.HiddenBackground
}

// StartInBackground re-execute the agent in the background.
// A hidden flag is appended to the command line
// to notify the child process that is already running in the background.
func (a *Agent) StartInBackground() error {
	args := os.Args[1:]
	args = append(args, "--hidden-background")
	//nolint:gosec
	c := exec.Command(args[0], args[1:]...)
	err := c.Start()
	if err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	return nil
}

// GenerateYAMLConfig returns the YAML configuration file corresponding to the currently running configuration.
func (a *Agent) GenerateYAMLConfig() (string, error) {
	c := a.cfg
	c.LocalSSHPassword = ""
	c.AgePubKey = ""
	c.GenerateConfig = false

	return cli.GenerateYAMLWithComments(*c)
}

// GetDNSCommand return the DNS command to perform SSH over DNS using a custom DNS.
func (a *Agent) GetDNSCommand() string {
	return a.cfg.CustomDNSCommand
}

// GetKillSwitchDays returns the configured killswitch.
func (a *Agent) GetKillSwitchDays() int {
	return a.cfg.KillSwitch
}

// Version returns the agent version.
func (a *Agent) Version() common.JVersion {
	return common.JSONVersion()
}
