package proxy

import (
	"crypto/tls"
)

// NewTLSConfig returns a new tls configuration
// this insecure configuration is required as the agent may need
// to be proxied on an internal HTTP proxy that performs TLS decryption.
func NewTLSConfig() *tls.Config {
	return &tls.Config{
		//nolint:gosec
		InsecureSkipVerify: true,
		//nolint:staticcheck // SA1019
		MinVersion: tls.VersionSSL30,
		NextProtos: []string{"http/1.1"},
	}
}
