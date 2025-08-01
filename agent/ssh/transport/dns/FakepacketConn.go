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

// MockAddr is a dummy net.Addr implementation
type MockAddr string

func (m MockAddr) Network() string { return "mock" }
func (m MockAddr) String() string  { return string(m) }

// FakePacketConn simulates a PacketConn using an internal channel
type FakePacketConn struct {
	closed        bool
	packets       chan packet
	localAddr     net.Addr
	deadline      time.Time
	currentPacket packet
}

func NewFakePacketConn(bufferSize int) *FakePacketConn {
	return &FakePacketConn{
		packets:   make(chan packet, bufferSize),
		localAddr: MockAddr("mock-local"),
		deadline:  time.Now().Add(time.Minute),
	}
}

func (m *FakePacketConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {

	pkt := <-m.packets
	n = copy(p, pkt.data)
	return n, pkt.addr, nil

}

func (m *FakePacketConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {

	if m.closed {
		return 0, errors.New("connection closed")
	}
	dataCopy := make([]byte, len(p))
	copy(dataCopy, p)
	m.packets <- packet{data: p, addr: addr}
	return len(p), nil
}

func (m *FakePacketConn) Close() error {
	if m.closed {
		return errors.New("already closed")
	}
	close(m.packets)
	m.closed = true
	return nil
}

func (m *FakePacketConn) LocalAddr() net.Addr {
	return m.localAddr
}

func (m *FakePacketConn) SetDeadline(t time.Time) error {
	m.deadline = t
	return nil
}

func (m *FakePacketConn) SetReadDeadline(t time.Time) error {
	m.deadline = t
	return nil
}

func (m *FakePacketConn) SetWriteDeadline(t time.Time) error {
	return nil // not implemented
}
