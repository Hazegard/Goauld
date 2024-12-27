package config

import (
	"Goauld/common/crypto"
	"fmt"
	"sync"
)

var serverOnce sync.Once
var srvCfg *ServerConfig

var privKey = ""

type ServerConfig struct {
	PrivKey           string
	HttpListenAddress string
	SshPort           int
}

func Get() *ServerConfig {
	serverOnce.Do(func() {
		srvCfg = &ServerConfig{
			PrivKey:           privKey,
			HttpListenAddress: ":3000",
			SshPort:           0,
		}
	})
	return srvCfg
}

func (s *ServerConfig) Decrypt(data []byte) (string, error) {
	return crypto.AsymDecrypt(s.PrivKey, data)
}

func (s *ServerConfig) LocalSShServer() string {
	return fmt.Sprintf("%s:%d", "127.0.0.1", s.SshPort)
}
