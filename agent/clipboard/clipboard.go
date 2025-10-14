package clipboard

import (
	"fmt"

	"github.com/atotto/clipboard"
)

func Copy() (string, error) {
	content, err := clipboard.ReadAll()
	if err != nil {
		return "", fmt.Errorf("error while reading from clipboard: %w", err)
	}

	return content, nil
}

func Paste(content string) error {
	return clipboard.WriteAll(content)
}
