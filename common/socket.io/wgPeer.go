package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/wireguard"
)

// newWGConfigEventMessage creates a new empty slice of ClipboardEvent messages.
func newWGConfigEventMessage() *wireguard.WGConfig {
	return &wireguard.WGConfig{}
}

// DecryptWGConfigEventMessage decrypts encrypted data into a slice of ClipboardEvent objects.
func DecryptWGConfigEventMessage(data []byte, c *crypto.SymCryptor) (wireguard.WGConfig, error) {
	decData, err := common.Decryptor[wireguard.WGConfig]{}.Decrypt(data, c, newWGConfigEventMessage)
	if err != nil {
		return wireguard.WGConfig{}, err
	}

	return *decData, err
}

// EncryptWGConfigEventMessage encrypts a slice of ClipboardEvent objects into bytes.
func EncryptWGConfigEventMessage(rpf wireguard.WGConfig, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(&rpf, c)
}

// NewEncryptedWGConfigEventMessage creates and encrypts a new ClipboardEvent message.
func NewEncryptedWGConfigEventMessage(rpf wireguard.WGConfig, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptWGConfigEventMessage(rpf, cryptor)
}
