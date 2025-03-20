package config

import (
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

	"Goauld/common/cli"

	"Goauld/common/crypto"
	"Goauld/common/ssh"

	"github.com/alecthomas/kong"
	"github.com/denisbrodbeck/machineid"
)

type Agent struct {
	Id                       string
	SShPrivateKey            string
	SharedSecret             string
	Cryptor                  *crypto.SymCryptor
	cfg                      *AgentConfig
	RemoteDynamicPortForward []int
	RemotePortForward        []int
	Platform                 string
	Architecture             string
	Username                 string
	Hostname                 string
	IPs                      []string
	Path                     string
}

var agent *Agent

// InitAgent parses the command lines arguments and initializes the temporary values (shared secret,etc...)
func InitAgent() (*kong.Context, error, []error) {
	warnings := []error{}
	// Parse the command line arguments
	ctx, cfg, err := parse()
	if err != nil {
		return nil, fmt.Errorf("parsing arguments: %v", err), nil
	}
	// Generate the shared secret
	if cfg.GenerateConfig {
		agent = &Agent{
			cfg: cfg,
		}
		return ctx, nil, warnings
	}
	sharedSecret, err := crypto.GeneratePassword(crypto.PasswordLength)
	if err != nil {
		return nil, fmt.Errorf("error generating ssh password: %v", err), nil
	}
	// Generate the local password if not provided
	if cfg.LocalSshPassword == "" {
		sshPassword, err := crypto.GeneratePassword(crypto.PasswordLength)
		if err != nil {
			return nil, fmt.Errorf("error generating ssh password: %v", err), nil
		}
		cfg.LocalSshPassword = sshPassword
	}

	// Generate the encryption mechanism
	cryptor, err := crypto.NewCryptor(sharedSecret)
	if err != nil {
		return nil, fmt.Errorf("initializing cryptor: %v", err), nil
	}

	// compute the agent ID used to identify it
	mid, err := machineid.ID()
	if err != nil {
		return nil, fmt.Errorf("error generating machine id: %v", err), nil
	}
	id := fmt.Sprintf("%x", md5.Sum([]byte(mid)))

	// Generate the agent name if not provided
	if cfg.Name == _name {
		name := ""
		userName, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error getting current user: %v", err), nil
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
		fmt.Println(userName.Name)

		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %v", err), nil
		}
		cfg.Name = fmt.Sprintf("%s@%s", name, hostname)
	}

	host, err := os.Hostname()
	if err != nil {
		warnings = append(warnings, fmt.Errorf("error getting hostname: %v", err))
	}

	u, err := user.Current()
	if err != nil {
		warnings = append(warnings, fmt.Errorf("error getting user: %v", err))
	}
	ips, errs := getIps()
	if len(errs) > 0 {
		warnings = append(warnings, fmt.Errorf("error getting ips: %v", errors.Join(errs...)))
	}

	currDir, err := os.Getwd()
	if err != nil {
		warnings = append(warnings, fmt.Errorf("error getting current directory: %v", err))
	}
	agent = &Agent{
		Id:                       id,
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
	}
	return ctx, nil, warnings
}

// Get return the Agent global object
func Get() *Agent {
	return agent
}

// Verbosity returns the current log verbosity
func (a *Agent) Verbosity() int {
	return a.cfg.Verbose
}

/*
// LocalSShdAddress returns the local address the sshd server listens
func (a *Agent) LocalSShdAddress() string {
	return net.JoinHostPort("127.0.0.1", strconv.Itoa(a.cfg.SshdPort))
}
*/
// LocalSShdPassword returns the current ssh password allowing to connect
func (a *Agent) LocalSShdPassword() string {
	return a.cfg.LocalSshPassword
}

/*
// LocalSshdPort returns the local SSHD password
func (a *Agent) LocalSshdPort() int {
	return a.cfg.SshdPort
}

// SetLocalSshdPort sets the SSHD port to the configuration
func (a *Agent) SetLocalSshdPort(p int) {
	a.cfg.SshdPort = p
}

// IsLocalSshdRandomPort returns whether the local SSHD port is random
func (a *Agent) IsLocalSshdRandomPort() bool {
	return a.cfg.SshdPort == 0
}
*/
// ControlSshServer returns the SSHD server
func (a *Agent) ControlSshServer() string {
	return a.cfg.SshServer
}

// IsRemoteForwardedSshdPortRandom returns whether the remote forwarded SSHD port is random
func (a *Agent) IsRemoteForwardedSshdPortRandom() bool {
	return a.cfg.RsshPort == 0
}

// RsshPort returns the remote forwarded SSHD port
func (a *Agent) RsshPort() int {
	return a.cfg.RsshPort
}

// RemoteForwardedSshdAddress returns the remote forwarded SSHD address
func (a *Agent) RemoteForwardedSshdAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.RsshPort)
}

// RemoteForwardedSocksAddress returns the remote forwarded Socks address
func (a *Agent) RemoteForwardedSocksAddress() string {
	return fmt.Sprintf("127.0.0.1:%d", a.cfg.SocksPort)
}

// RemoteForwardedSocksPort returns the remote forwarded Socks port
func (a *Agent) RemoteForwardedSocksPort() int {
	return a.cfg.SocksPort
}

// RemoteForwardedSocksPort returns the remote forwarded Socks port
func (a *Agent) RemoteForwardedSshdPort() int {
	return a.cfg.RsshPort
}

// UpdateSocksPort set the new socks port
func (a *Agent) UpdateSocksPort(port int) {
	a.cfg.SocksPort = port
}

// UpdateSshdPort set the new socks port
func (a *Agent) UpdateSshdPort(port int) {
	a.cfg.RsshPort = port
}

// ServerUrl return the HTTP control server URL
func (a *Agent) ServerUrl() string {
	url := ""
	if strings.HasPrefix(a.cfg.Server, "http://") {
		url = a.cfg.Server
	} else if strings.HasPrefix(a.cfg.Server, "https://") {
		url = a.cfg.Server
	} else {
		url = "http://" + a.cfg.Server
	}

	return url
}

// WSshUrl return the SSH over Websocket connection URL
func (a *Agent) WSshUrl() string {
	return fmt.Sprintf("%s/wssh/%s", a.ServerUrl(), a.Id)
}

// SocketIoUrl return the control connection URL
func (a *Agent) SocketIoUrl() string {
	return fmt.Sprintf("%s/socket.io/", a.ServerUrl())
}

// SSHTTPUrl return the SSH over HTTP connection URL
func (a *Agent) SSHTTPUrl() string {
	return fmt.Sprintf("%s/sshttp/%s", a.ServerUrl(), a.Id)
}

// TlsUrl return the SSH over TLS connection URL
func (a *Agent) TlsUrl() string {
	url, err := url.Parse(a.cfg.TlsServer)
	if err != nil {
		// An error occured while parsing the TLS sever, so we pass it as is to see if the TLS connection can succeed
		return a.cfg.TlsServer
	}
	if url.Port() == "" {
		return fmt.Sprintf("%s:443", a.cfg.TlsServer)
	}
	return a.cfg.TlsServer
}

// Name returns the agent name
func (a *Agent) Name() string {
	return a.cfg.Name
}

// GetKeepalive returns the duration between two keepalive ping
func (a *Agent) GetKeepalive() time.Duration {
	return time.Duration(a.cfg.KeepAlive)
}

// GetRsshOrder returns the order thtat the agent should follow to attempt to connect
// to the SSHD server
func (a *Agent) GetRsshOrder() []string {
	return a.cfg.RsshOrder
}

// AgePubKey returns the age public key used to encrypt asymetrically data
func (a *Agent) AgePubKey() string {
	return a.cfg.AgePubKey
}

// SshdEnabled returns whether the sshd server is enabled
func (a *Agent) SshdEnabled() bool {
	return a.cfg.Sshd
}

// SocksEnabled returns whether the socks server is enabled
func (a *Agent) SocksEnabled() bool {
	return a.cfg.Socks || a.cfg.SocksPort != 0
}

// SocksUseSystemProxy returns whether the agent Socks proxy should use the system proxy (if applicable)
func (a *Agent) SocksUseSystemProxy() bool {
	return a.cfg.SocksUseSystemProxy
}

// AddSshdToRpf adds the SSHD conf to the Remote port forwarding list
func (a *Agent) AddSshdToRpf() {
	sshdRpf := ssh.RemotePortForwarding{
		ServerPort: a.cfg.RsshPort,
		AgentPort:  -1,
		AgentIP:    "0.0.0.0",
		Tag:        "sshd",
	}
	a.cfg.RemotePortForwarding = append(a.cfg.RemotePortForwarding, sshdRpf)
}

// NoProxy return whether the agent should ignore the potential system proxy
func (a *Agent) NoProxy() bool {
	return a.cfg.NoProxy
}

// Proxy returns the proxy provided by the configuration
func (a *Agent) Proxy() *url.URL {
	return a.cfg.Proxy
}

// AddSocksToRpf adds the SSHD conf to the Remote port forwarding list
func (a *Agent) AddSocksToRpf() {
	sshdRpf := ssh.RemotePortForwarding{
		ServerPort: a.cfg.SocksPort,
		AgentPort:  -1,
		AgentIP:    "0.0.0.0",
		Tag:        "socks",
	}
	a.cfg.RemotePortForwarding = append(a.cfg.RemotePortForwarding, sshdRpf)
}

// GetRemotePortForwarding returns the configured remote port forwarding
func (a *Agent) GetRemotePortForwarding() []ssh.RemotePortForwarding {
	return a.cfg.RemotePortForwarding
}

// func getID() string {
// 	id, err != machin
// }

// getIps returns the IP on the hosts, excluding local network addresses
func getIps() ([]string, []error) {
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

// GetMexRetries returns the configured max retries before killing the agent
func (a *Agent) GetMexRetries() uint {
	return uint(a.cfg.MaxRetries)
}

// DoGenerateConfig return whether the configuration generation should is enabled
func (a *Agent) DoGenerateConfig() bool {
	return a.cfg.GenerateConfig
}

// ShouldRunInBackground returns whether the agent should be run in background
func (a *Agent) ShouldRunInBackground() bool {
	return a.cfg.Background && !a.cfg.HiddenBackground
}

// StartInBackground re-execute the agent in background. A hidden flag is appended to the command line
// In order to notify the child process that is already running in background
func (a *Agent) StartInBackground() error {
	args := append(os.Args, "--hidden-background")
	c := exec.Command(args[0], args[1:]...)
	err := c.Start()
	if err != nil {
		return fmt.Errorf("failed to start agent: %s", err)
	}
	return nil
}

// GenerateYAMLConfig returns the yaml configuration file corresponding to the currently running configuration
func (a *Agent) GenerateYAMLConfig() (string, error) {
	c := a.cfg
	c.LocalSshPassword = ""
	c.AgePubKey = ""
	c.GenerateConfig = false
	return cli.GenerateYAMLWithComments(*c)
}
