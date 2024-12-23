package ssh

import (
	"crypto"
	"fmt"
	gossh "golang.org/x/crypto/ssh"

	"crypto/rand"
	"encoding/pem"
	"golang.org/x/crypto/ed25519"
)

// ParseSSHPublicKey parses an SSH public key string and returns an ssh.PublicKey type.
func ParseSSHPublicKey(publicKeyStr string) (gossh.PublicKey, error) {
	// Convert the string to bytes
	keyBytes := []byte(publicKeyStr)

	// Parse the public key
	publicKey, _, _, _, err := gossh.ParseAuthorizedKey(keyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SSH public key: %w", err)
	}

	return publicKey, nil
}

// GenerateEd25519SSHKey generates an Ed25519 SSH key pair and returns the private and public keys as strings.
func GenKey() (privateKeyPEM string, publicKeySSH string, err error) {
	// Generate an Ed25519 key pair
	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate Ed25519 key: %v", err)
	}
	p, err := gossh.MarshalPrivateKey(crypto.PrivateKey(privateKey), "")
	if err != nil {
		panic(err)
	}
	privateKeyPem := pem.EncodeToMemory(p)
	privateKeyString := string(privateKeyPem)
	fmt.Println(privateKeyPEM)
	// Generate the corresponding public key in OpenSSH format
	publicKey, err := gossh.NewPublicKey(privateKey.Public())
	if err != nil {
		return "", "", fmt.Errorf("failed to generate public key: %v", err)
	}
	publicKeySSH = string(gossh.MarshalAuthorizedKey(publicKey))

	return privateKeyString, publicKeySSH, nil
}
