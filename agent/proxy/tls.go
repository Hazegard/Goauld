package proxy

import (
	"crypto/tls"
)

// NewTlsConfig returns a new tls configuration
// this insecure configuration is required as the agent may need
// to be proxied on an internal HTTP proxy that performs TLS decryption
func NewTlsConfig() *tls.Config {
	return &tls.Config{
		InsecureSkipVerify: true,
		//nolint:staticcheck
		MinVersion: tls.VersionSSL30,
		NextProtos: []string{"http/1.1"},
	}
}
