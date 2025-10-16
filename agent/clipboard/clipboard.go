package clipboard

import (
	"context"
	"fmt"

	"github.com/gopasspw/clipboard"
)

func Copy(ctx context.Context) (string, error) {
	content, err := clipboard.ReadAll(ctx)
	if err != nil {
		return "", fmt.Errorf("error while reading from clipboard: %w", err)
	}

	return string(content), nil
}

func Paste(ctx context.Context, content string) error {
	return clipboard.WriteAll(ctx, []byte(content))
}
