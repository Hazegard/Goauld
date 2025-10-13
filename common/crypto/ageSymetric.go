package crypto

import (
	"bytes"
	"fmt"
	"io"

	"filippo.io/age"
)

// SymCryptor handle the symmetrical encryption.
type SymCryptor struct {
	key       string
	identity  *age.ScryptIdentity
	recipient *age.ScryptRecipient
}

// NewCryptor returns a new SymCryptor.
func NewCryptor(key string) (*SymCryptor, error) {
	identity, err := age.NewScryptIdentity(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scrypt identity: %w", err)
	}
	recipient, err := age.NewScryptRecipient(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create Scrypt recipient: %w", err)
	}
	c := &SymCryptor{
		key:       key,
		identity:  identity,
		recipient: recipient,
	}

	return c, nil
}

// Encrypt encrypts the data using the shared secret key.
func (c *SymCryptor) Encrypt(data []byte) ([]byte, error) {
	// Create a reader for the encrypted data
	var encryptedBuffer bytes.Buffer

	// Create a decryption reader
	writer, err := age.Encrypt(&encryptedBuffer, c.recipient)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	// Write the plaintext data to the encryption writer
	_, err = writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to write plaintext: %w", err)
	}

	// Close the writer to finalize encryption
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %w", err)
	}

	return encryptedBuffer.Bytes(), nil
}

// Decrypt decrypts the data using the shared secret key.
func (c *SymCryptor) Decrypt(data []byte) ([]byte, error) {
	// Create a reader for the encrypted data
	encryptedBuffer := bytes.NewReader(data)

	// Create a decryption reader
	reader, err := age.Decrypt(encryptedBuffer, c.identity)
	if err != nil {
		return nil, fmt.Errorf("failed to create decryption reader: %w", err)
	}

	// Read the decrypted message
	decryptedMessage, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read decrypted message: %w", err)
	}

	return decryptedMessage, nil
}
