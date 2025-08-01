package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type PasswordValidationRequest struct {
	HashPassword string `json:"hash_password"`
	EventId      string `json:"eventId"`
}

func newPasswordValidationRequest() *PasswordValidationRequest {
	return &PasswordValidationRequest{}
}

func DecryptPasswordValidationRequest(data []byte, c *crypto.SymCryptor) (*PasswordValidationRequest, error) {
	a, err := common.Decryptor[PasswordValidationRequest]{}.Decrypt(data, c, newPasswordValidationRequest)
	return a, err
}

func EncryptPasswordValidationRequest(agent *PasswordValidationRequest, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptPasswordValidationRequest(password string, eventId string, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &PasswordValidationRequest{
		HashPassword: password,
		EventId:      eventId,
	}
	return EncryptPasswordValidationRequest(message, cryptor)
}

type PasswordValidationResponse struct {
	Response bool `json:"response"`
}

func newPasswordValidationResponse() *PasswordValidationResponse {
	return &PasswordValidationResponse{}
}

func DecryptPasswordValidationResponse(data []byte, c *crypto.SymCryptor) (*PasswordValidationResponse, error) {
	a, err := common.Decryptor[PasswordValidationResponse]{}.Decrypt(data, c, newPasswordValidationResponse)
	return a, err
}

func EncryptPasswordValidationResponse(agent *PasswordValidationResponse, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptPasswordValidationResponse(response bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &PasswordValidationResponse{
		Response: response,
	}
	return EncryptPasswordValidationResponse(message, cryptor)
}
