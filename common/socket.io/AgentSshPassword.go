package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

const SendAgentSshPasswordEvent = "SSH Password"
const SendAgentSshPasswordError = "SSH Password error"
const SendAgentSshPasswordSuccess = "SSH Password success"

// AgentSshPasswordMessage holds the ssh password used to authenticate on the agent
type AgentSshPasswordMessage struct {
	AgentSshPassword string `json:"ssh_password"`
}

func newAgentSshPasswordMessage() *AgentSshPasswordMessage {
	return &AgentSshPasswordMessage{}
}

func DecryptAgentSshPasswordMessage(data []byte, c *crypto.SymCryptor) (*AgentSshPasswordMessage, error) {
	agent, err := common.Decryptor[AgentSshPasswordMessage]{}.Decrypt(data, c, newAgentSshPasswordMessage)
	return agent, err
}

func EncryptAgentSshPasswordMessage(agent *AgentSshPasswordMessage, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptedAgentSshPasswordMessage(privateKey string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &AgentSshPasswordMessage{AgentSshPassword: privateKey}
	return EncryptAgentSshPasswordMessage(message, cryptor)
}
