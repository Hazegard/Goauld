package wireguard

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"strings"
)

type PublicKeyHex string

func (p PublicKeyHex) String() string {
	return string(p)
}

type PrivateKeyHex string

func (p PrivateKeyHex) String() string {
	return string(p)
}

type PublicKey string

func (p PublicKey) String() string {
	return string(p)
}
func (p PublicKey) Hex() (PublicKeyHex, error) {
	return getHexPublicKey(p)
}

type PrivateKey string

func (p PrivateKey) String() string {
	return string(p)
}
func (p PrivateKey) Hex() (PrivateKeyHex, error) {
	return getHexPrivateKey(p)
}

type WGConfig struct {
	PublicKey  PublicKey  `default:"" json:"wg-public-key" omitempty:"" name:"wg-public-key" yaml:"wg-public-key" optional:"" help:"Wireguard configuration file."`
	PrivateKey PrivateKey `default:"" json:"wg-private-key" omitempty:"" name:"wg-private-key" yaml:"wg-private-key" optional:"" help:"Wireguard configuration file."`
	IP         string     `default:"" json:"wg-ip" name:"wg-ip" omitempty:"" yaml:"wg-ip" optional:"" help:"Wireguard configuration file."`
}

func getHexPublicKey(pubKey PublicKey) (PublicKeyHex, error) {
	p, err := toHexKey(pubKey)
	if err != nil {
		return "", err
	}

	return PublicKeyHex(p), nil
}

func getHexPrivateKey(privKey PrivateKey) (PrivateKeyHex, error) {
	p, err := toHexKey(privKey)
	if err != nil {
		return "", err
	}

	return PrivateKeyHex(p), nil
}

// toHexKey ensures a WireGuard key is returned in hex form.
// If the input is base64, it's converted to hex.
// If it's already hex, it's returned unchanged.
func toHexKey[T PublicKey | PrivateKey](key T) (string, error) {
	k := strings.TrimSpace(string(key))

	// Try to decode as base64
	data, err := base64.StdEncoding.DecodeString(k)
	if err == nil && len(data) == 32 {
		// Valid base64 key (WireGuard keys are always 32 bytes)
		return hex.EncodeToString(data), nil
	}

	// If it's already hex and valid length, just return
	data, err = hex.DecodeString(k)
	if err == nil && len(data) == 32 {
		return k, nil
	}

	return "", errors.New("invalid key format (not valid base64 or hex WireGuard key)")
}
