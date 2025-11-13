package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// IsAliveRequest represents a request containing a hashed password and event ID.
type IsAliveRequest struct {
	EventID string `json:"eventId"`
}

// newIsAliveRequest creates a new empty IsAliveRequest.
func newIsAliveRequest() *IsAliveRequest {
	return &IsAliveRequest{}
}

// DecryptIsAliveRequest decrypts encrypted request data into a IsAliveRequest.
func DecryptIsAliveRequest(data []byte, c *crypto.SymCryptor) (*IsAliveRequest, error) {
	a, err := common.Decryptor[IsAliveRequest]{}.Decrypt(data, c, newIsAliveRequest)

	return a, err
}

// EncryptIsAliveRequest encrypts a IsAliveRequest into bytes.
func EncryptIsAliveRequest(agent *IsAliveRequest, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptIsAliveRequest creates and encrypts a new IsAliveRequest.
func NewEncryptIsAliveRequest(eventID string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &IsAliveRequest{
		EventID: eventID,
	}

	return EncryptIsAliveRequest(message, cryptor)
}

// IsAliveResponse represents a response indicating if password validation succeeded.
type IsAliveResponse struct {
	Response bool `json:"response"`
}

// newIsAliveResponse creates a new empty IsAliveResponse.
func newIsAliveResponse() *IsAliveResponse {
	return &IsAliveResponse{}
}

// DecryptIsAliveResponse decrypts encrypted response data into a IsAliveResponse.
func DecryptIsAliveResponse(data []byte, c *crypto.SymCryptor) (*IsAliveResponse, error) {
	a, err := common.Decryptor[IsAliveResponse]{}.Decrypt(data, c, newIsAliveResponse)

	return a, err
}

// EncryptIsAliveResponse encrypts a IsAliveResponse into bytes.
func EncryptIsAliveResponse(agent *IsAliveResponse, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptIsAliveResponse creates and encrypts a new IsAliveResponse.
func NewEncryptIsAliveResponse(response bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &IsAliveResponse{
		Response: response,
	}

	return EncryptIsAliveResponse(message, cryptor)
}
