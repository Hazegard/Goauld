package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// ExitRequest holds the inforamtion regarding the exit requested by the server.
type ExitRequest struct {
	Kill           bool   `json:"kill"`
	Delete         bool   `json:"delete"`
	HashedPassword []byte `json:"password"`
}

// newExitRequest creates a new Exit request.
func newExitRequest() *ExitRequest {
	return &ExitRequest{}
}

// DecryptExitRequest decrypts an exit request.
func DecryptExitRequest(data []byte, c *crypto.SymCryptor) (*ExitRequest, error) {
	a, err := common.Decryptor[ExitRequest]{}.Decrypt(data, c, newExitRequest)

	return a, err
}

// EncryptExitRequest encrypts an exit request.
func EncryptExitRequest(agent *ExitRequest, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptExitRequest returns a Exit request.
func NewEncryptExitRequest(kill bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &ExitRequest{
		Kill: kill,
	}

	return EncryptExitRequest(message, cryptor)
}

// ExitResponse holds the information regarding the exit requested by the server.
type ExitResponse struct {
	Response bool `json:"response"`
}

// newExitResponse creates a new Exit response.
func newExitResponse() *ExitResponse {
	return &ExitResponse{}
}

// DecryptExitResponse returns a decrypted Exit response.
func DecryptExitResponse(data []byte, c *crypto.SymCryptor) (*ExitResponse, error) {
	a, err := common.Decryptor[ExitResponse]{}.Decrypt(data, c, newExitResponse)

	return a, err
}

// EncryptExitResponse returns an encrypted Exit response.
func EncryptExitResponse(agent *ExitResponse, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptExitResponse returns an Exit Response.
func NewEncryptExitResponse(response bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &ExitResponse{
		Response: response,
	}

	return EncryptExitResponse(message, cryptor)
}
