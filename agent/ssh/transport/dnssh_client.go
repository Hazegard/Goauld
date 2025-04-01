package transport

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"www.bamsoftware.com/git/dnstt.git/dns"
	"www.bamsoftware.com/git/dnstt.git/turbotunnel"
)

type DNSSH struct {
	udpConn net.PacketConn
	pconn   net.PacketConn
	session *smux.Session
	Stream  *smux.Stream
	kcpConn *kcp.UDPSession
}

// smux streams will be closed after this much time without receiving data.
const idleTimeout = 2 * time.Minute

// dnsNameCapacity returns the number of bytes remaining for encoded data after
// including domain in a DNS name.
func dnsNameCapacity(domain dns.Name) int {
	// Names must be 255 octets or shorter in total length.
	// https://tools.ietf.org/html/rfc1035#section-2.3.4
	capacity := 255
	// Subtract the length of the null terminator.
	capacity -= 1
	for _, label := range domain {
		// Subtract the length of the label and the length octet.
		capacity -= len(label) + 1
	}
	// Each label may be up to 63 bytes long and requires 64 bytes to
	// encode.
	capacity = capacity * 63 / 64
	// Base32 expands every 5 bytes to 8.
	capacity = capacity * 5 / 8
	return capacity
}

func run( /*pubkey []byte,*/ domain dns.Name, remoteAddr net.Addr, pconn net.PacketConn) (*kcp.UDPSession, *smux.Session, *smux.Stream, error) {

	mtu := dnsNameCapacity(domain) - 8 - 1 - numPadding - 1 // clientid + padding length prefix + padding + data length prefix
	if mtu < 80 {
		return nil, nil, nil, fmt.Errorf("domain %s leaves only %d bytes for payload", domain, mtu)
	}

	log.Get().Trace().Str("Mode", "DNSSH").Msgf("effective MTU %d", mtu)

	// Open a KCP conn on the PacketConn.
	conn, err := kcp.NewConn2(remoteAddr, nil, 0, 0, pconn)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("opening KCP conn: %v", err)
	}
	defer func() {
		log.Printf("end session %08x", conn.GetConv())
		conn.Close()
	}()
	log.Trace().Str("Mode", "DNSSH").Msgf("opening session %08x", conn.GetConv())
	// Permit coalescing the payloads of consecutive sends.
	conn.SetStreamMode(true)
	// Disable the dynamic congestion window (limit only by the maximum of
	// local and remote static windows).
	conn.SetNoDelay(
		0, // default nodelay
		0, // default interval
		0, // default resend
		1, // nc=1 => congestion window off
	)
	conn.SetWindowSize(turbotunnel.QueueSize/2, turbotunnel.QueueSize/2)
	if rc := conn.SetMtu(mtu); !rc {
		return nil, nil, nil, fmt.Errorf("setting mtu failed")
	}

	// Put a Noise channel on top of the KCP conn.
	// rw, err := noise.NewClient(conn, pubkey)
	// if err != nil {
	// 	return nil, nil, nil, fmt.Errorf("error creating noise client: %v", err)
	// }

	// Start a smux session on the Noise channel.
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 2
	smuxConfig.KeepAliveTimeout = idleTimeout
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024 // default is 65536
	sess, err := smux.Client( /*rw*/ conn, smuxConfig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error opening smux session: %v", err)
	}

	stream, err := sess.OpenStream()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("error opening stream: %v", err)
	}
	return conn, sess, stream, nil
}

func NewDNSSH() (*DNSSH, error) {
	// noisepubkey := config.Get().Id
	// pubkey, err := noise.DecodeKey(noisepubkey)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "pubkey format error: %v\n", err)
	// 	os.Exit(1)
	// }
	domain, err := dns.ParseName(config.Get().DNSServer())
	if err != nil {
		fmt.Fprintf(os.Stderr, "invalid domain %+q: %v\n", flag.Arg(0), err)
		os.Exit(1)
	}

	// Iterate over the remote resolver address options and select one and
	// only one.
	var remoteAddr net.Addr
	var udpConn net.PacketConn

	remoteAddr, err = net.ResolveUDPAddr("udp", config.Get().DNSServer())
	udpConn, err = net.ListenUDP("udp", nil)

	pconn := NewDNSPacketConn(udpConn, remoteAddr, domain)
	kcpConn, sess, stream, err := run( /*pubkey,*/ domain, remoteAddr, pconn)
	if err != nil {
		return nil, fmt.Errorf("error initializing DNS tunnel: %s", err)
	}
	return &DNSSH{
		udpConn: udpConn,
		pconn:   pconn,
		session: sess,
		Stream:  stream,
		kcpConn: kcpConn,
	}, nil
}

func (d *DNSSH) Read(b []byte) (n int, err error) {
	return d.Stream.Read(b)
}
func (d *DNSSH) Write(b []byte) (n int, err error) {
	return d.Stream.Write(b)
}

func (d *DNSSH) LocalAddr() net.Addr {
	return d.Stream.LocalAddr()
}
func (d *DNSSH) RemoteAddr() net.Addr {
	return d.Stream.RemoteAddr()
}
func (d *DNSSH) SetDeadline(t time.Time) error {
	return d.Stream.SetDeadline(t)
}

func (d *DNSSH) SetReadDeadline(t time.Time) error {
	return d.Stream.SetReadDeadline(t)
}
func (d *DNSSH) SetWriteDeadline(t time.Time) error {
	return d.Stream.SetWriteDeadline(t)
}
func (d *DNSSH) Close() error {
	var errs []error
	errs = append(errs, d.kcpConn.Close())
	errs = append(errs, d.session.Close())
	errs = append(errs, d.udpConn.Close())
	errs = append(errs, d.pconn.Close())
	return errors.Join(errs...)
}
