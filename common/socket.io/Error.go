package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type SioError struct {
	Message string `json:"message"`
	Code    string `json:"code"`
}

func newSioErrorMessage() *SioError {
	return &SioError{}
}

func DecryptSioErrorMessage(data []byte, c *crypto.SymCryptor) (*SioError, error) {
	a, err := common.Decryptor[SioError]{}.Decrypt(data, c, newSioErrorMessage)
	return a, err
}

func EncryptSioErrorMessage(err *SioError, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(err, c)
}

func NewEncryptedSioErrorMessage(err *SioError, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptSioErrorMessage(err, cryptor)
}
