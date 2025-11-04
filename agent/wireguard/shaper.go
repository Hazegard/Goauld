package wireguard

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

type LimitReader struct {
	r       io.Reader
	limiter *rate.Limiter
	ctx     context.Context
}

// NewLimitReader returns a reader that implements io.Reader with rate limiting.
func NewLimitReader(r io.Reader, limiter *rate.Limiter) *LimitReader {
	return &LimitReader{
		r:       r,
		limiter: limiter,
		ctx:     context.Background(),
	}
}

// NewReaderWithContext returns a reader that implements io.Reader with rate limiting.
func NewReaderWithContext(r io.Reader, ctx context.Context) *LimitReader {
	return &LimitReader{
		r:   r,
		ctx: ctx,
	}
}

// Read reads bytes into p.
func (s *LimitReader) Read(p []byte) (int, error) {
	if s.limiter == nil {
		return s.r.Read(p)
	}
	n, err := s.r.Read(p)
	if err != nil {
		return n, err
	}
	if err := s.limiter.WaitN(s.ctx, n); err != nil {
		return n, err
	}

	return n, nil
}
