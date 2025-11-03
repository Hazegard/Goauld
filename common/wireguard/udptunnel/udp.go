package udptunnel

import (
	"context"
	"encoding/gob"
	"io"
)

type UdpPacket struct {
	Src     string
	Payload []byte
}

func init() {
	gob.Register(&UdpPacket{})
}

// UdpChannel encodes/decodes udp payloads over a stream.
type UdpChannel struct {
	R *gob.Decoder
	W *gob.Encoder
	C io.Closer
}

func (o *UdpChannel) Encode(src string, b []byte) error {
	return o.W.Encode(UdpPacket{
		Src:     src,
		Payload: b,
	})
}

func (o *UdpChannel) Decode(p *UdpPacket) error {
	return o.R.Decode(p)
}

func isDone(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
