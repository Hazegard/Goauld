package clipboard

import (
	"context"
	"fmt"

	"github.com/gopasspw/clipboard"
)

// Copy returns the content of the agent clipboard.
func Copy(ctx context.Context) ([]byte, error) {
	content, err := clipboard.ReadAll(ctx)
	if err != nil {
		return nil, fmt.Errorf("error while reading from clipboard: %w", err)
	}

	return content, nil
}

// Paste set the agent clipboard content.
func Paste(ctx context.Context, content []byte) error {
	return clipboard.WriteAll(ctx, content)
}
