package crypto

import (
	"bytes"
	"fmt"
	"io"

	"filippo.io/age"
)

type SymCryptor struct {
	key       string
	identity  *age.ScryptIdentity
	recipient *age.ScryptRecipient
}

func NewCryptor(key string) (*SymCryptor, error) {
	identity, err := age.NewScryptIdentity(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scrypt identity: %v", err)
	}
	recipient, err := age.NewScryptRecipient(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scrypt recipient: %v", err)
	}
	c := &SymCryptor{
		key:       key,
		identity:  identity,
		recipient: recipient,
	}
	return c, nil
}

func (c *SymCryptor) Encrypt(data []byte) ([]byte, error) {
	// Create a reader for the encrypted data
	var encryptedBuffer bytes.Buffer

	// Create a decryption reader
	writer, err := age.Encrypt(&encryptedBuffer, c.recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %v", err)
	}

	// Write the plaintext data to the encryption writer
	_, err = io.WriteString(writer, string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to write plaintext: %v", err)
	}

	// Close the writer to finalize encryption
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %v", err)
	}
	return encryptedBuffer.Bytes(), nil
}

func (c *SymCryptor) Decrypt(data []byte) ([]byte, error) {
	// Create a reader for the encrypted data
	encryptedBuffer := bytes.NewReader(data)

	// Create a decryption reader
	reader, err := age.Decrypt(encryptedBuffer, c.identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %v", err)
	}

	// Read the decrypted message
	decryptedMessage, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted message: %v", err)
	}

	return decryptedMessage, nil
}
