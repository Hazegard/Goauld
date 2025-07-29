package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

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
