package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// ClipboardMessage represents a request containing a hashed password and event ID.
type ClipboardMessage struct {
	Content      string `json:"content"`
	HashPassword string `json:"hash_password"`
	Error        bool   `json:"error"`
}

// newClipboardMessageEventMessage creates a new empty slice of ClipboardMessageEvent messages.
func newClipboardMessageEventMessage() *ClipboardMessage {
	return &ClipboardMessage{}
}

// DecryptClipboardMessageEventMessage decrypts encrypted data into a slice of ClipboardMessageEvent objects.
func DecryptClipboardMessageEventMessage(data []byte, c *crypto.SymCryptor) (ClipboardMessage, error) {
	decData, err := common.Decryptor[ClipboardMessage]{}.Decrypt(data, c, newClipboardMessageEventMessage)
	if err != nil {
		return ClipboardMessage{}, err
	}

	return *decData, err
}

// EncryptClipboardMessageEventMessage encrypts a slice of ClipboardMessageEvent objects into bytes.
func EncryptClipboardMessageEventMessage(rpf ClipboardMessage, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(&rpf, c)
}

// NewEncryptedClipboardMessageEventMessage creates and encrypts a new ClipboardMessageEvent message.
func NewEncryptedClipboardMessageEventMessage(rpf ClipboardMessage, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptClipboardMessageEventMessage(rpf, cryptor)
}

// ClipboardRequestMessage represents a request containing a hashed password and event ID.
type ClipboardRequestMessage struct {
	EventID      string `json:"eventId"`
	HashPassword string `json:"hash_password"`
}

// newClipboardMessageEventMessage creates a new empty slice of ClipboardMessageEvent messages.
func newClipboardRequestMessageEventMessage() *ClipboardRequestMessage {
	return &ClipboardRequestMessage{}
}

// DecryptClipboardRequestMessageEventMessage decrypts encrypted data into a slice of ClipboardMessageEvent objects.
func DecryptClipboardRequestMessageEventMessage(data []byte, c *crypto.SymCryptor) (ClipboardRequestMessage, error) {
	decData, err := common.Decryptor[ClipboardRequestMessage]{}.Decrypt(data, c, newClipboardRequestMessageEventMessage)
	if err != nil {
		return ClipboardRequestMessage{}, err
	}

	return *decData, err
}

// EncryptClipboardRequestMessageEventMessage encrypts a slice of ClipboardMessageEvent objects into bytes.
func EncryptClipboardRequestMessageEventMessage(rpf ClipboardRequestMessage, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(&rpf, c)
}

// NewEncryptedClipboardRequestMessageEventMessage creates and encrypts a new ClipboardMessageEvent message.
func NewEncryptedClipboardRequestMessageEventMessage(rpf ClipboardRequestMessage, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptClipboardRequestMessageEventMessage(rpf, cryptor)
}
