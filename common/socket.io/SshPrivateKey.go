package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// SSHPrivateKeyMessage represents a message containing an SSH private key.
type SSHPrivateKeyMessage struct {
	SSHPrivateKey string `json:"ssh_public_key"`
}

// newSSHPublicKeyMessage creates a new empty SSHPrivateKeyMessage.
func newSSHPublicKeyMessage() *SSHPrivateKeyMessage {
	return &SSHPrivateKeyMessage{}
}

// DecryptSSHPrivateKeyMessage decrypts encrypted data into a SSHPrivateKeyMessage.
func DecryptSSHPrivateKeyMessage(data []byte, c *crypto.SymCryptor) (*SSHPrivateKeyMessage, error) {
	agent, err := common.Decryptor[SSHPrivateKeyMessage]{}.Decrypt(data, c, newSSHPublicKeyMessage)

	return agent, err
}

// EncryptSSHPrivateKeyMessage encrypts a SSHPrivateKeyMessage into bytes.
func EncryptSSHPrivateKeyMessage(agent *SSHPrivateKeyMessage, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptedSSHPrivateKeyMessage creates and encrypts a new SSHPrivateKeyMessage.
func NewEncryptedSSHPrivateKeyMessage(privateKey string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &SSHPrivateKeyMessage{SSHPrivateKey: privateKey}

	return EncryptSSHPrivateKeyMessage(message, cryptor)
}
