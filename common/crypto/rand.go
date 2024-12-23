package crypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

const (
	PasswordLength = 69                                                                                         // Desired password length
	charset        = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()-_=+[]{}|:,.<>?/" // Allowed characters
)

// GeneratePassword generates a secure random password of a given length
func GeneratePassword(length int) (string, error) {
	if length <= 0 {
		return "", fmt.Errorf("password length must be greater than zero")
	}

	password := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	// Generate random characters for the password
	for i := 0; i < length; i++ {
		randomIndex, err := rand.Int(rand.Reader, charsetLen) // Generate a secure random index
		if err != nil {
			return "", fmt.Errorf("failed to generate random index: %v", err)
		}
		password[i] = charset[randomIndex.Int64()]
	}

	return string(password), nil
}
