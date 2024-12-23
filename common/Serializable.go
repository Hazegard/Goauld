package common

import (
	"Goauld/common/crypto"
	"encoding/json"
	"fmt"
)

// Ptr constraining a type to its pointer type
type Ptr[T any] interface {
	*T
}

// the first type param will match pointer serTypes and infer U
type Decryptor[U any] struct{}

func (f Decryptor[U]) Decrypt(data []byte, c *crypto.SymCryptor, init func() *U) (*U, error) {
	// declare var of non-pointer type. this is not nil!
	var a = init()

	d, err := c.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("error decrypting data: %s", err)
	}

	err = json.Unmarshal(d, a)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling data: %s", err)
	}
	// address it and convert to pointer type (still not nil)
	return a, nil
}

func Encrypt[T any](t *T, c *crypto.SymCryptor) ([]byte, error) {
	data, err := json.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("error marshaling data: %s", err)
	}
	enc, err := c.Encrypt(data)
	if err != nil {
		return nil, fmt.Errorf("error encrypting data: %s", err)
	}
	return enc, nil
}
