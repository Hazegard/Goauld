package blind

import (
	"Goauld/common/log"
	"Goauld/common/net/blind"
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"net"
	"strings"
	"time"

	"github.com/miekg/dns"
)

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

// DNSClient represents a DNS tunnel client
type DNSClient struct {
	//listenAddr string
	dnsServer string
	sessionID string
	tld       string
	dnsClient *dns.Client
	debug     bool
}

func generateSessionID() string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ234567"
	result := make([]byte, blind.SessionIDLength)
	for i := range result {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		result[i] = chars[n.Int64()]
	}
	return string(result)
}

// NewDNSClient creates a new DNS tunnel client
func NewDNSClient(dnsServer, dnsDomain string, debug bool) (*DNSClient, error) {
	sessionID := generateSessionID()

	dnsClient := &dns.Client{
		Net:          "udp",
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}

	return &DNSClient{
		//listenAddr: listenAddr,
		dnsServer: dnsServer,
		sessionID: sessionID,
		tld:       dnsDomain,
		dnsClient: dnsClient,
		debug:     debug,
	}, nil
}

// Add a new method to reset client state
func (c *DNSClient) resetState() {
	// Generate new session ID for new connections
	c.sessionID = blind.GenerateSessionID()

	if c.debug {
		log.Printf("Reset client state with new session ID: %s", c.sessionID)
	}
}

func (c *DNSClient) Tunnel(conn net.Conn) {
	c.resetState()
	c.handleConnection(conn)
}

/*
// Update Start method to handle multiple connections
func (c *DNSClient) Start() error {
	listener, err := net.Listen("tcp", c.listenAddr)
	if err != nil {
		return fmt.Errorf("failed to start TCP listener: %v", err)
	}
	defer listener.Close()

	if c.debug {
		log.Printf("TCP listener started on %s", c.listenAddr)
		log.Printf("Tunneling to DNS server at %s", c.dnsServer)
	}

	for {
		// Reset state for each new connection
		c.resetState()

		if c.debug {
			log.Printf("Waiting for new connection with session ID: %s", c.sessionID)
		}

		conn, err := listener.Accept()
		if err != nil {
			if c.debug {
				log.Printf("Error accepting connection: %v", err)
			}
			continue
		}

		if c.debug {
			log.Printf("New connection accepted, handling with session ID: %s", c.sessionID)
		}

		// Handle connection in goroutine
		go func() {
			c.handleConnection(conn)
			if c.debug {
				log.Printf("Connection handled, ready for next connection")
			}
		}()
	}
}
*/
// Update handleConnection to be more robust
func (c *DNSClient) handleConnection(conn net.Conn) {
	defer conn.Close()

	done := make(chan struct{})
	defer close(done)

	errChan := make(chan error, 2)

	// Start read goroutine
	go func() {
		buffer := make([]byte, blind.MaxChunkSize)
		sequence := uint16(0)
		for {
			select {
			case <-done:
				return
			default:
				n, err := conn.Read(buffer)
				if err != nil {
					if err != io.EOF && !strings.Contains(err.Error(), "use of closed network connection") {
						if c.debug {
							log.Printf("Error reading from connection: %v", err)
						}
					}
					errChan <- err
					return
				}
				if n > 0 {
					if err := c.sendChunk(buffer[:n], sequence); err != nil {
						if c.debug {
							log.Printf("Error sending chunk: %v", err)
						}
						errChan <- err
						return
					}
					sequence++
				}
			}
		}
	}()

	// Start poll goroutine
	go func() {
		for {
			select {
			case <-done:
				return
			default:
				data, err := c.pollForData()
				if err != nil {
					if c.debug {
						log.Printf("Poll error: %v", err)
					}
					errChan <- err
					return
				}
				if data != nil {
					if string(data) == "CLOSED" {
						if c.debug {
							log.Printf("Server indicated session closed")
						}
						errChan <- fmt.Errorf("session closed by server")
						return
					}
					if len(data) > 0 && string(data) != "EMPTY" {
						if _, err := conn.Write(data); err != nil {
							if c.debug {
								log.Printf("Error writing to connection: %v", err)
							}
							errChan <- err
							return
						}
						if c.debug {
							log.Printf("Wrote %d bytes from poll to local connection", len(data))
						}
					}
				}
				time.Sleep(blind.PollDelay)
			}
		}
	}()

	// Wait for either an error or done signal
	select {
	case err := <-errChan:
		if c.debug {
			log.Printf("Session ended: %v", err)
		}
	case <-done:
	}
}

// sendChunk sends a chunk of data through DNS
func (c *DNSClient) sendChunk(chunk []byte, sequence uint16) error {
	// Split large chunks into smaller ones
	maxChunkSize := 100 // Reduced chunk size

	chunks := blind.SplitDataIntoChunks(chunk, maxChunkSize)

	for i, subChunk := range chunks {
		encodedData := blind.EncodeDNSSafe(subChunk)

		// Construct FQDN
		fqdn := fmt.Sprintf("%s.%04x.%s.%s",
			encodedData,
			sequence+uint16(i),
			c.sessionID,
			c.tld)

		if c.debug {
			log.Printf("=== Sending DNS Query ===")
			log.Printf("To: %s", c.dnsServer)
			log.Printf("FQDN: %s", fqdn)
			log.Printf("Sequence: %d", sequence+uint16(i))
			log.Printf("Chunk size: %d", len(subChunk))
		}

		_, err := c.sendQuery(fqdn)
		if err != nil {
			return fmt.Errorf("failed to send chunk %d: %v", sequence+uint16(i), err)
		}
	}

	return nil
}

// sendQuery sends a DNS query and returns the response
func (c *DNSClient) sendQuery(fqdn string) ([]byte, error) {
	msg := new(dns.Msg)
	msg.SetQuestion(dns.Fqdn(fqdn), dns.TypeTXT)
	msg.RecursionDesired = true

	// Set EDNS0 options for larger responses
	opt := new(dns.OPT)
	opt.Hdr.Name = "."
	opt.Hdr.Rrtype = dns.TypeOPT
	opt.SetUDPSize(4096)
	msg.Extra = append(msg.Extra, opt)

	for attempt := 1; attempt <= blind.MaxRetries; attempt++ {
		if c.debug {
			log.Printf("Attempt %d of %d", attempt, blind.MaxRetries)
		}

		r, _, err := c.dnsClient.Exchange(msg, c.dnsServer)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				if c.debug {
					log.Printf("Query failed: %v, retrying...", err)
				}
				time.Sleep(blind.RetryDelay)
				continue
			}
			return nil, err
		}

		if r.Rcode != dns.RcodeSuccess {
			if c.debug {
				log.Printf("Query returned error code %d, retrying...", r.Rcode)
			}
			time.Sleep(blind.RetryDelay)
			continue
		}

		if len(r.Answer) > 0 {
			if txt, ok := r.Answer[0].(*dns.TXT); ok {
				responseText := strings.Join(txt.Txt, "")
				if responseText == "EMPTY" {
					return nil, nil
				}

				decodedResponse, err := blind.DecodeDNSSafe(responseText)
				if err != nil {
					if c.debug {
						log.Printf("Failed to decode response: %v", err)
					}
					return nil, err
				}
				return decodedResponse, nil
			}
		}

		return nil, nil
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// pollForData polls the server for available data
func (c *DNSClient) pollForData() ([]byte, error) {
	fqdn := fmt.Sprintf("AA.ffff.%s.%s", c.sessionID, c.tld)

	if c.debug {
		log.Printf("=== Sending Poll Query ===")
		log.Printf("To: %s", c.dnsServer)
		log.Printf("FQDN: %s", fqdn)
	}

	response, err := c.sendQuery(fqdn)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "EMPTY" {
		return nil, nil
	}

	return response, nil
}

// sendData sends data through DNS
func (c *DNSClient) sendData(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Start with sequence 0
	sequence := uint16(0)

	// Send data in chunks
	return c.sendChunk(data, sequence)
}
