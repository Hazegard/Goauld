package config

import (
	"Goauld/common/crypto"
	"sync"
)

var serverOnce sync.Once
var srvCfg *ServerConfig

var privKey = ""

type ServerConfig struct {
	PrivKey           string
	HttpListenAddress string
	SshPort           string
}

func Get() *ServerConfig {
	serverOnce.Do(func() {
		srvCfg = &ServerConfig{
			PrivKey:           privKey,
			HttpListenAddress: ":3000",
			SshPort:           "22",
		}
	})
	return srvCfg
}

func (s *ServerConfig) Decrypt(data []byte) (string, error) {
	return crypto.AsymDecrypt(s.PrivKey, data)
}
