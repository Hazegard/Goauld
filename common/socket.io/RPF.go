package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/ssh"
)

// newRemotePortForwardingMessage creates a new empty slice of RemotePortForwarding messages.
func newRemotePortForwardingMessage() *[]ssh.RemotePortForwarding {
	return &[]ssh.RemotePortForwarding{}
}

// DecryptRemotePortForwardingMessage decrypts encrypted data into a slice of RemotePortForwarding objects.
func DecryptRemotePortForwardingMessage(data []byte, c *crypto.SymCryptor) ([]ssh.RemotePortForwarding, error) {
	decData, err := common.Decryptor[[]ssh.RemotePortForwarding]{}.Decrypt(data, c, newRemotePortForwardingMessage)
	if err != nil {
		return nil, err
	}

	return *decData, err
}

// EncryptRemotePortForwardingMessage encrypts a slice of RemotePortForwarding objects into bytes.
func EncryptRemotePortForwardingMessage(rpf []ssh.RemotePortForwarding, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(&rpf, c)
}

// NewEncryptedRemotePortForwardingMessage creates and encrypts a new RemotePortForwarding message.
func NewEncryptedRemotePortForwardingMessage(rpf []ssh.RemotePortForwarding, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptRemotePortForwardingMessage(rpf, cryptor)
}
