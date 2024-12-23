package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
)

type Agent struct {
	Id string
}

func newAgent() *Agent {
	return &Agent{}
}

func DecryptAgent(data []byte, c *crypto.SymCryptor) (*Agent, error) {
	agent, err := common.Decryptor[Agent]{}.Decrypt(data, c, newAgent)
	return agent, err
}

func EncryptAgent(agent *Agent, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(agent, c)
}
