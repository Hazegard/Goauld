// Package dns holds the SSH over DNS client
package dns

import (
	"errors"
	"net"
	"time"
)

type packet struct {
	data []byte
	addr net.Addr
}

// MockAddr is a dummy net.Addr implementation.
type MockAddr string

// Network mock.
func (m MockAddr) Network() string { return "mock" }
func (m MockAddr) String() string  { return string(m) }

// FakePacketConn simulates a PacketConn using an internal channel.
type FakePacketConn struct {
	closed    bool
	packets   chan packet
	localAddr net.Addr
	deadline  time.Time
}

// NewFakePacketConn returns a new FakePacketConn.
func NewFakePacketConn(bufferSize int) *FakePacketConn {
	return &FakePacketConn{
		packets:   make(chan packet, bufferSize),
		localAddr: MockAddr("mock-local"),
		deadline:  time.Now().Add(time.Minute),
	}
}

// ReadFrom waits for a packet in the channel and copies it to the destination buffer.
func (m *FakePacketConn) ReadFrom(p []byte) (int, net.Addr, error) {
	pkt := <-m.packets
	n := copy(p, pkt.data)

	return n, pkt.addr, nil
}

// WriteTo writes to the dest channel.
func (m *FakePacketConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if m.closed {
		return 0, errors.New("connection closed")
	}
	dataCopy := make([]byte, len(p))
	copy(dataCopy, p)
	m.packets <- packet{data: p, addr: addr}

	return len(p), nil
}

// Close closes.
func (m *FakePacketConn) Close() error {
	if m.closed {
		return errors.New("already closed")
	}
	close(m.packets)
	m.closed = true

	return nil
}

// LocalAddr dummy local address.
func (m *FakePacketConn) LocalAddr() net.Addr {
	return m.localAddr
}

// SetDeadline sets the deadline.
func (m *FakePacketConn) SetDeadline(t time.Time) error {
	m.deadline = t

	return nil
}

// SetReadDeadline set the read deadline.
func (m *FakePacketConn) SetReadDeadline(t time.Time) error {
	m.deadline = t

	return nil
}

// SetWriteDeadline noop.
func (m *FakePacketConn) SetWriteDeadline(_ time.Time) error {
	return nil // not implemented
}
