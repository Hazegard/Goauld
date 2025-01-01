package crypto

import (
	"bytes"
	"filippo.io/age"
	"fmt"
	"io"
)

// ageAsymetric handle the asymetric cryptography using the age library

// AsymEncrypt encrypt the plaintext using the provided age public key
func AsymEncrypt(publicKey string, plainText string) ([]byte, error) {
	// Parse the recipient's public key
	recipients, err := age.ParseX25519Recipient(publicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %v", err)
	}

	// Create a buffer to write encrypted data
	var encryptedBuffer bytes.Buffer

	// Create a new age encryptor targeting the recipient
	writer, err := age.Encrypt(&encryptedBuffer, recipients)
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %v", err)
	}

	// Write the plaintext to the encryptor
	if _, err := io.WriteString(writer, plainText); err != nil {
		return nil, fmt.Errorf("failed to write plaintext: %v", err)
	}

	// Close the writer to finalize encryption
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize encryption: %v", err)
	}

	// Return the encrypted data
	return encryptedBuffer.Bytes(), nil
}

// AsymDecrypt decrypts the encrypted data using the recipient's private key.
func AsymDecrypt(privateKey string, encryptedData []byte) (string, error) {
	// Parse the recipient's private key
	identity, err := age.ParseX25519Identity(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to parse private key: %v", err)
	}

	// Create a buffer to hold the encrypted data
	encryptedBuffer := bytes.NewReader(encryptedData)

	// Create a new age decryptor using the private key (identity)
	reader, err := age.Decrypt(encryptedBuffer, identity)
	if err != nil {
		return "", fmt.Errorf("failed to create decryptor: %v", err)
	}

	// Read the decrypted data
	var decryptedBuffer bytes.Buffer
	_, err = io.Copy(&decryptedBuffer, reader)
	if err != nil {
		return "", fmt.Errorf("failed to read decrypted data: %v", err)
	}

	// Return the decrypted plaintext as a string
	return decryptedBuffer.String(), nil
}
