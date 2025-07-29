package socket_io

import (
	"strings"

	"Goauld/agent/config"
	"Goauld/common"
	"Goauld/common/crypto"
)

// AgentData holds the ssh password used to authenticate on the agent
type AgentData struct {
	AgentSshPassword string `json:"ssh_password"`
	Platform         string `json:"platform"`
	Architecture     string `json:"architecture"`
	Username         string `json:"username"`
	Hostname         string `json:"hostname"`
	IPs              string `json:"ips"`
	Path             string `json:"path"`
	HasStaticPwd     bool   `json:"has_static_pwd"`
}

func newAgentSshPasswordMessage() *AgentData {
	return &AgentData{}
}

func DecryptAgentSshPasswordMessage(data []byte, c *crypto.SymCryptor) (*AgentData, error) {
	a, err := common.Decryptor[AgentData]{}.Decrypt(data, c, newAgentSshPasswordMessage)
	return a, err
}

func EncryptAgentSshPasswordMessage(agent *AgentData, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptedAgentSshPasswordMessage(a *config.Agent, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &AgentData{
		AgentSshPassword: a.LocalSSHDPassword(),
		Platform:         a.Platform,
		Architecture:     a.Architecture,
		Username:         a.Username,
		Hostname:         a.Hostname,
		IPs:              strings.Join(a.IPs, ","),
		Path:             a.Path,
		HasStaticPwd:     a.PrivateSshdPassword() != "",
	}
	return EncryptAgentSshPasswordMessage(message, cryptor)
}
