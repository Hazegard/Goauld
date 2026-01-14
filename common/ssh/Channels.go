package ssh

import (
	"encoding/binary"
	"errors"
	"io"
)

type Channel struct {
	rw io.ReadWriter
}

func NewChannel(rw io.ReadWriter) *Channel {
	return &Channel{rw: rw}
}

// Response format:
// uint32 status (1 = ok, 0 = fail)
// uint32 payload_len
// payload

// WriteResponse serialize the response and send it over SSH channel.
func (c *Channel) WriteResponse(ok bool, payload []byte) error {
	var status uint32
	if ok {
		status = 1
	}

	if err := binary.Write(c.rw, binary.BigEndian, status); err != nil {
		return err
	}

	//nolint:gosec
	l := uint32(len(payload))
	if err := binary.Write(c.rw, binary.BigEndian, l); err != nil {
		return err
	}

	if l > 0 {
		_, err := c.rw.Write(payload)

		return err
	}

	return nil
}

// ReadResponse unserialize the data received through the SSH channel.
func (c *Channel) ReadResponse() ([]byte, error) {
	var status uint32
	if err := binary.Read(c.rw, binary.BigEndian, &status); err != nil {
		return nil, err
	}

	var l uint32
	if err := binary.Read(c.rw, binary.BigEndian, &l); err != nil {
		return nil, err
	}

	buf := make([]byte, l)
	if l > 0 {
		if _, err := io.ReadFull(c.rw, buf); err != nil {
			return nil, err
		}
	}

	if status != 1 {
		return nil, errors.New("remote operation failed")
	}

	return buf, nil
}
