package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type ChunkedAgent struct {
	Chunk     int
	LastChunk int
	Data      []byte
}

// newSSHPublicKeyMessage creates a new empty ChunkedAgent.
func newChunkedAgent() *ChunkedAgent {
	return &ChunkedAgent{}
}

// DecryptChunkedData decrypts encrypted data into a ChunkedAgent.
func DecryptChunkedData(data []byte, c *crypto.SymCryptor) (*ChunkedAgent, error) {
	agent, err := common.Decryptor[ChunkedAgent]{}.Decrypt(data, c, newChunkedAgent)

	return agent, err
}

// EncryptChunkedAgent encrypts a ChunkedAgent into bytes.
func EncryptChunkedAgent(agent *ChunkedAgent, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptedChunkedAgent creates and encrypts a new ChunkedAgent.
func NewEncryptedChunkedAgent(chunk int, last int, data []byte, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &ChunkedAgent{
		Chunk:     chunk,
		LastChunk: last,
		Data:      data,
	}

	return EncryptChunkedAgent(message, cryptor)
}
