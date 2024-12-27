package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type Register struct {
	Id        string `json:"id"`
	Name      []byte `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

const RegisterEvent = "register"
const RegisterError = "register error"
const RegisterSuccess_AskSSHPassword = "register success"

type Deregister struct {
}

const DeregisterEvent = "deregister"
const DeregisterError = "deregister error"
const DeregisterSuccess = "deregister success"

const Connect = "connect"
const ConnectError = "connect error"
const ConnectSuccess = "connect error"

const Disconnect = "connect"
const DisconnectError = "connect error"
const DisconnectSuccess = "connect error"

type DisconnectMessage struct{}

const SendSshPrivateKeyEvent = "SSH Public Key"
const SendSshHPrivateKeyError = "SSH Public Key error"
const SendSshPrivateKeySuccess = "SSH Public Key success"

type SshPrivateKeyMessage struct {
	SshPrivateKey string `json:"ssh_public_key"`
}

func newSshPublicKeyMessage() *SshPrivateKeyMessage {
	return &SshPrivateKeyMessage{}
}

func DecryptSshPrivateKeyMessage(data []byte, c *crypto.SymCryptor) (*SshPrivateKeyMessage, error) {
	agent, err := common.Decryptor[SshPrivateKeyMessage]{}.Decrypt(data, c, newSshPublicKeyMessage)
	return agent, err
}

func EncryptSshPrivateKeyMessage(agent *SshPrivateKeyMessage, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptedSshPrivateKeyMessage(privateKey string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &SshPrivateKeyMessage{SshPrivateKey: privateKey}
	return EncryptSshPrivateKeyMessage(message, cryptor)
}

const SendAgentSshPasswordEvent = "SSH Password"
const SendAgentSshPasswordError = "SSH Password error"
const SendAgentSshPasswordSuccess = "SSH Password success"

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

const PingEvent = "Ping"
const PingError = "Ping error"
const PingSuccess = "Ping success"

const PongEvent = "Pong"
const PongError = "Pong error"
const PongSuccess = "Pong error"
