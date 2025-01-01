package proxy

import (
	"crypto/tls"
)

// NewTlsConfig returns a new tls configuration
// this insecure configuration is required as the agent may need
// to be proxified on internal HTTP proxy that performs TLS decryption
func NewTlsConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		MinVersion:         tls.VersionSSL30,
		NextProtos:         []string{"http/1.1"},
	}
}
