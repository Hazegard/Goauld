//go:build mini

package pwgen

import (
	"errors"
)

func GetXKCDPassword() (string, error) {
	return "", errors.New("not implemented in mini stager")
}
