package ttycommon

import (
	"io"
	"maze.io/x/ttyrec"
	"time"
)

// Encoder can write chunks of bytes in a ttyrec format.
type Encoder struct {
	w io.Writer

	// started indicates if we have started writing
	started bool

	// startedAt is the time of the first write
	startedAt time.Time
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w: w,
	}
}

func (e *Encoder) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}

	header := ttyrec.Header{Len: uint32(len(p))}
	if !e.started {
		e.started = true
		e.startedAt = time.Now()

		header.Time.Set(time.Since(time.Unix(0, 0)))
	} else {
		header.Time.Set(time.Since(time.Unix(0, 0)))
	}

	// Write header.
	if _, err := header.WriteTo(e.w); err != nil {
		return 0, err
	}

	// Write data.
	return e.w.Write(p)
}
