//go:build !mini

// Package config holds the agent configuration
package config

import (
	"Goauld/common/cli"
	"Goauld/common/crypto/pwgen"
	"Goauld/common/log"
	"Goauld/common/wireguard"
	"net"
	"time"

	//nolint:gosec
	"crypto/md5"
	"errors"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strings"

	"Goauld/common/crypto"
	"Goauld/common/ssh"

	"github.com/alecthomas/kong"
	"github.com/keygen-sh/machineid"
)

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

// PrivateSshdPassword return the static password.
func (a *Agent) PrivateSshdPassword() string {
	if a.cfg.PrivatePassword != "" {
		return a.cfg.PrivatePassword
	}
	if _private_password != "" {
		return _private_password
	}
	if a.cfg.DisablePassword {
		return ""
	}
	p, err := pwgen.GetXKCDPassword()
	if err != nil {
		p, err = crypto.GeneratePassword(30)
		if err != nil {
			panic(err)
		}
	}
	a.IsStaticPasswordDynamic = true
	_private_password = p

	return _private_password
}

// ValidatePassword return whether the incoming password is valid.
// The password is concatenated using the dynamic password, and the compiled-time-defined password
// (private sshd password).
func (a *Agent) ValidatePassword(in string) bool {
	return in == a.PrivateSshdPassword()+a.LocalSSHDPassword()
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

// GenerateYAMLConfig returns the YAML configuration file corresponding to the currently running configuration.
func (a *Agent) GenerateYAMLConfig() (string, error) {
	c := a.cfg
	c.LocalSSHPassword = ""
	c.AgePubKey = ""
	c.GenerateConfig = false

	return cli.GenerateYAMLWithComments(*c)
}

// WGEnabled returns whether the wireguard server is enabled.
func (a *Agent) WGEnabled() bool {
	return a.cfg.WG
}

// DisableWG returns whether the http proxy server is enabled.
func (a *Agent) DisableWG() {
	a.cfg.WG = false
}
func (a *Agent) GenerateWireguardConfig() error {
	pri, pub, err := wireguard.GenerateWireGuardKeyPair()
	if err != nil {
		return err
	}
	a.Wireguard = wireguard.WGConfig{
		PublicKey:  pub,
		PrivateKey: pri,
		IP:         "100.64.0.1",
	}

	return nil
}
