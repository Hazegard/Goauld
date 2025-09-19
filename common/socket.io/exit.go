package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type ExitRequest struct {
	Kill           bool   `json:"kill"`
	Delete         bool   `json:"delete"`
	HashedPassword []byte `json:"password"`
}

func newExitRequest() *ExitRequest {
	return &ExitRequest{}
}

func DecryptExitRequest(data []byte, c *crypto.SymCryptor) (*ExitRequest, error) {
	a, err := common.Decryptor[ExitRequest]{}.Decrypt(data, c, newExitRequest)
	return a, err
}

func EncryptExitRequest(agent *ExitRequest, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptExitRequest(kill bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &ExitRequest{
		Kill: kill,
	}
	return EncryptExitRequest(message, cryptor)
}

type ExitResponse struct {
	Response bool `json:"response"`
}

func newExitResponse() *ExitResponse {
	return &ExitResponse{}
}

func DecryptExitResponse(data []byte, c *crypto.SymCryptor) (*ExitResponse, error) {
	a, err := common.Decryptor[ExitResponse]{}.Decrypt(data, c, newExitResponse)
	return a, err
}

func EncryptExitResponse(agent *ExitResponse, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}

func NewEncryptExitResponse(response bool, cryptor *crypto.SymCryptor) ([]byte, error) {
	message := &ExitResponse{
		Response: response,
	}
	return EncryptExitResponse(message, cryptor)
}
