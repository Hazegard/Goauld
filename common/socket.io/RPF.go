package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/ssh"
)

const (
	SendRemotePortForwardingDataEvent   = "RemotePortForwarding Data"
	SendRemotePortForwardingDataError   = "RemotePortForwarding Data error"
	SendRemotePortForwardingDataSuccess = "RemotePortForwarding Data success"
)

func newRemotePortForwardingMessage() *[]ssh.RemotePortForwarding {
	return &[]ssh.RemotePortForwarding{}
}

func DecryptRemotePortForwardingMessage(data []byte, c *crypto.SymCryptor) ([]ssh.RemotePortForwarding, error) {
	decData, err := common.Decryptor[[]ssh.RemotePortForwarding]{}.Decrypt(data, c, newRemotePortForwardingMessage)
	if err != nil {
		return nil, err
	}

	return *decData, err
}

func EncryptRemotePortForwardingMessage(rpf []ssh.RemotePortForwarding, c *crypto.SymCryptor) ([]byte, error) {
	return common.Encrypt(&rpf, c)
}

func NewEncryptedRemotePortForwardingMessage(err []ssh.RemotePortForwarding, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptRemotePortForwardingMessage(err, cryptor)
}
