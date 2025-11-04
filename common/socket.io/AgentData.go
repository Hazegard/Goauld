// Package socketio holds the common socket.io information
package socketio

import (
	"strings"

	"Goauld/agent/config"
	"Goauld/common"
	"Goauld/common/crypto"
)

// AgentData holds agent information and SSH credentials for authentication.
type AgentData struct {
	AgentSSHPassword string          `json:"ssh_password"`
	Platform         string          `json:"platform"`
	Architecture     string          `json:"architecture"`
	Username         string          `json:"username"`
	Hostname         string          `json:"hostname"`
	IPs              string          `json:"ips"`
	Path             string          `json:"path"`
	HasStaticPwd     bool            `json:"has_static_pwd"`
	AgentVersion     common.JVersion `json:"agent_version"`
	WireguardPubKey  string          `json:"wireguard_pub_key"`
	WireguardIP      string          `json:"wireguard_ip"`
}

// newAgentSSHPasswordMessage creates a new empty AgentData instance.
func newAgentSSHPasswordMessage() *AgentData {
	return &AgentData{}
}

// DecryptAgentSSHPasswordMessage decrypts encrypted data into an AgentData object.
func DecryptAgentSSHPasswordMessage(data []byte, c *crypto.SymCryptor) (*AgentData, error) {
	a, err := common.Decryptor[AgentData]{}.Decrypt(data, c, newAgentSSHPasswordMessage)

	return a, err
}

// EncryptAgentSSHPasswordMessage encrypts an AgentData object into bytes.
func EncryptAgentSSHPasswordMessage(agent *AgentData, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptedAgentSSHPasswordMessage creates and encrypts a new AgentData message from config.Agent.
func NewEncryptedAgentSSHPasswordMessage(a *config.Agent, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &AgentData{
		AgentSSHPassword: a.LocalSSHDPassword(),
		Platform:         a.Platform,
		Architecture:     a.Architecture,
		Username:         a.Username,
		Hostname:         a.Hostname,
		IPs:              strings.Join(a.IPs, ","),
		Path:             a.Path,
		HasStaticPwd:     a.PrivateSshdPassword() != "",
		AgentVersion:     a.Version(),
		WireguardPubKey:  a.Wireguard.PublicKey.String(),
		WireguardIP:      a.Wireguard.IP,
	}

	return EncryptAgentSSHPasswordMessage(message, cryptor)
}
