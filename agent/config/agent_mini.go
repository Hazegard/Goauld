//go:build mini
// +build mini

package config

import "Goauld/common/crypto"

// getIPs returns the IP on the hosts, excluding local network addresses.
func getIPs() ([]string, []error) {

	return nil, nil
}

// PrivateSshdPassword return the static password.
func (a *Agent) PrivateSshdPassword() string {
	return ""
}

var agent *Agent

// Get returns the Agent global object.
func Get() *Agent {
	return agent
}

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
}
