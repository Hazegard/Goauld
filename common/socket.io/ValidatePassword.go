package socketio

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

// PasswordValidationRequest represents a request containing a hashed password and event ID.
type PasswordValidationRequest struct {
	HashPassword string `json:"hash_password"`
	EventID      string `json:"eventId"`
}

// newPasswordValidationRequest creates a new empty PasswordValidationRequest.
func newPasswordValidationRequest() *PasswordValidationRequest {
	return &PasswordValidationRequest{}
}

// DecryptPasswordValidationRequest decrypts encrypted request data into a PasswordValidationRequest.
func DecryptPasswordValidationRequest(data []byte, c *crypto.SymCryptor) (*PasswordValidationRequest, error) {
	a, err := common.Decryptor[PasswordValidationRequest]{}.Decrypt(data, c, newPasswordValidationRequest)

	return a, err
}

// EncryptPasswordValidationRequest encrypts a PasswordValidationRequest into bytes.
func EncryptPasswordValidationRequest(agent *PasswordValidationRequest, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptPasswordValidationRequest creates and encrypts a new PasswordValidationRequest.
func NewEncryptPasswordValidationRequest(password string, eventID string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &PasswordValidationRequest{
		HashPassword: password,
		EventID:      eventID,
	}

	return EncryptPasswordValidationRequest(message, cryptor)
}

// PasswordValidationResponse represents a response indicating if password validation succeeded.
type PasswordValidationResponse struct {
	Response bool `json:"response"`
}

// newPasswordValidationResponse creates a new empty PasswordValidationResponse.
func newPasswordValidationResponse() *PasswordValidationResponse {
	return &PasswordValidationResponse{}
}

// DecryptPasswordValidationResponse decrypts encrypted response data into a PasswordValidationResponse.
func DecryptPasswordValidationResponse(data []byte, c *crypto.SymCryptor) (*PasswordValidationResponse, error) {
	a, err := common.Decryptor[PasswordValidationResponse]{}.Decrypt(data, c, newPasswordValidationResponse)

	return a, err
}

// EncryptPasswordValidationResponse encrypts a PasswordValidationResponse into bytes.
func EncryptPasswordValidationResponse(agent *PasswordValidationResponse, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

// NewEncryptPasswordValidationResponse creates and encrypts a new PasswordValidationResponse.
func NewEncryptPasswordValidationResponse(response bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &PasswordValidationResponse{
		Response: response,
	}

	return EncryptPasswordValidationResponse(message, cryptor)
}
