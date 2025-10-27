package ttyencoder

import (
	"fmt"
	"io"
	"os"

	"github.com/hazegard/togettyc/ttycommon"
	"github.com/klauspost/compress/zstd"
)

func NewEncoder() *Encoder {
	return &Encoder{}
}

type Encoder struct {
	*ttycommon.Encoder
	f io.WriteCloser

	append   bool
	compress bool
	dst      string
}

func (e *Encoder) Close() error {
	return e.f.Close()
}

func (e *Encoder) WithCompress(compress bool) *Encoder {
	e.compress = compress
	return e
}

func (e *Encoder) WithAppend(append bool) *Encoder {
	e.append = append
	return e
}

func (e *Encoder) WithOutput(dst string) *Encoder {
	e.dst = dst
	return e
}

func (e *Encoder) Open() (*Encoder, error) {
	f, err := openEncoder(e.append, e.compress, e.dst)

	if err != nil {
		return nil, fmt.Errorf("error opening recorder: %v", err)
	}
	e.f = f
	e.Encoder = ttycommon.NewEncoder(f)
	return e, nil
}

func openEncoder(append bool, compress bool, output string) (io.WriteCloser, error) {
	var (
		f   *os.File
		err error
	)

	if append {
		f, err = os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0666)
	} else {
		f, err = os.Create(output)
	}
	if err != nil {
		return nil, err
	}

	if compress {
		zstdFile, err := zstd.NewWriter(f)
		if err != nil {
			return nil, err
		}
		return zstdFile, nil
	} else {
		return f, nil
	}
}
