package socket_io

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/ssh"
)

const SendRemotePortForwardingDataEvent = "RemotePortForwarding Data"
const SendRemotePortForwardingDataError = "RemotePortForwarding Data error"
const SendRemotePortForwardingDataSuccess = "RemotePortForwarding Data success"

func newRemotePortForwardingMessage() *[]string {
	return &[]string{}
}

func DecryptRemotePortForwardingMessage(data []byte, c *crypto.SymCryptor) ([]ssh.RemotePortForwarding, error) {
	decData, err := common.Decryptor[[]string]{}.Decrypt(data, c, newRemotePortForwardingMessage)
	if err != nil {
		return nil, err
	}
	rpfs := make([]ssh.RemotePortForwarding, 0)
	for _, d := range *decData {
		rpf := ssh.RemotePortForwarding{}
		err := rpf.UnmarshalText([]byte(d))
		if err != nil {
			return nil, err
		}
		rpfs = append(rpfs, rpf)
	}
	return rpfs, err
}

func EncryptRemotePortForwardingMessage(rpf []ssh.RemotePortForwarding, c *crypto.SymCryptor) ([]byte, error) {
	rpfString := []string{}
	for _, v := range rpf {
		rpfString = append(rpfString, v.Info())
	}
	return common.Encrypt(&rpfString, c)
}

func NewEncryptedRemotePortForwardingMessage(err []ssh.RemotePortForwarding, cryptor *crypto.SymCryptor) ([]byte, error) {
	return EncryptRemotePortForwardingMessage(err, cryptor)
}
