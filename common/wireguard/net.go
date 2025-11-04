package wireguard

import (
	"fmt"

	"golang.zx2c4.com/wireguard/wgctrl/wgtypes"
)

// GenerateWireGuardKeyPair uses the WireGuard API to create a key pair.
func GenerateWireGuardKeyPair() (PrivateKey, PublicKey, error) {
	privKey, err := wgtypes.GeneratePrivateKey()
	if err != nil {
		return "", "", fmt.Errorf("failed to generate private key: %w", err)
	}

	pubKey := PublicKey(privKey.PublicKey().String())

	return PrivateKey(privKey.String()), pubKey, nil
}
