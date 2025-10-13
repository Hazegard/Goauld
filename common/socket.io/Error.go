package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// SioError represents a Socket.IO error message with a code and description.
type SioError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

// newSioErrorMessage creates a new empty SioError instance.
func newSioErrorMessage() *SioError {
	return &SioError{}
}

// DecryptSioErrorMessage decrypts encrypted data into a SioError object.
func DecryptSioErrorMessage(data []byte, c *crypto.SymCryptor) (*SioError, error) {
	a, err := common.Decryptor[SioError]{}.Decrypt(data, c, newSioErrorMessage)

	return a, err
}

// EncryptSioErrorMessage encrypts a SioError object into bytes.
func EncryptSioErrorMessage(err *SioError, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(err, c)
}

// NewEncryptedSioErrorMessage creates and encrypts a new SioError message.
func NewEncryptedSioErrorMessage(err *SioError, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptSioErrorMessage(err, cryptor)
}
