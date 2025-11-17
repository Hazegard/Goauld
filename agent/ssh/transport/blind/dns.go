package blind

import (
	"Goauld/common/log"
	"Goauld/common/net/blind"
	"crypto/rand"
	"errors"
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

// DNSClient represents a DNS tunnel client.
type DNSClient struct {
	// listenAddr string
	dnsServer string
	sessionID string
	agentID   string
	tld       string
	dnsClient *dns.Client
	mode      string
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

// NewDNSClient creates a new DNS tunnel client.
func NewDNSClient(dnsServer []string, dnsDomain string, mode string, agentID string) (*DNSClient, error) {
	sessionID := generateSessionID()
	switch {
	case strings.EqualFold(mode, "control"):
		sessionID = "C" + sessionID[1:]
	case strings.EqualFold(mode, "ssh"):
		sessionID = "S" + sessionID[1:]
	}

	dnsClient := &dns.Client{
		Net:          "udp",
		ReadTimeout:  2 * time.Second,
		WriteTimeout: 2 * time.Second,
	}
	var server string
	for _, srv := range dnsServer {
		if TestDNSServer(srv, dnsDomain) {
			server = srv
		}
	}

	return &DNSClient{
		dnsServer: server,
		sessionID: sessionID,
		agentID:   agentID,
		tld:       dnsDomain,
		dnsClient: dnsClient,
		mode:      mode,
	}, nil
}

// Add a new method to reset client state.
func (c *DNSClient) resetState() {
	// Generate new session ID for new connections
	sessionID := blind.GenerateSessionID()
	switch {
	case strings.EqualFold(c.mode, "control"):
		sessionID = "C" + sessionID[1:]
	case strings.EqualFold(c.mode, "ssh"):
		sessionID = "S" + sessionID[1:]
	}
	c.sessionID = sessionID

	log.Debug().Msgf("Reset client state with new session ID: %s", c.sessionID)
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
		log.Debug().Msgf("TCP listener started on %s", c.listenAddr)
		log.Debug().Msgf("Tunneling to DNS server at %s", c.dnsServer)
	}

	for {
		// Reset state for each new connection
		c.resetState()

		if c.debug {
			log.Debug().Msgf("Waiting for new connection with session ID: %s", c.sessionID)
		}

		conn, err := listener.Accept()
		if err != nil {
			if c.debug {
				log.Debug().Msgf("Error accepting connection: %v", err)
			}
			continue
		}

		if c.debug {
			log.Debug().Msgf("New connection accepted, handling with session ID: %s", c.sessionID)
		}

		// Handle connection in goroutine
		go func() {
			c.handleConnection(conn)
			if c.debug {
				log.Debug().Msgf("Connection handled, ready for next connection")
			}
		}()
	}
}
*/
// Update handleConnection to be more robust.
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
					if !errors.Is(err, io.EOF) && !strings.Contains(err.Error(), "use of closed network connection") {
						log.Debug().Msgf("Error reading from connection: %v", err)
					}
					errChan <- err

					return
				}
				if n > 0 {
					if err := c.sendChunk(buffer[:n], sequence); err != nil {
						log.Debug().Msgf("Error sending chunk: %v", err)
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
		time.Sleep(2 * time.Second)
		c.register()
		for {
			select {
			case <-done:
				return
			default:
				data, err := c.pollForData()
				if err != nil {
					log.Debug().Msgf("Poll error: %v", err)
					errChan <- err

					return
				}
				if data != nil {
					if string(data) == "CLOSED" {
						log.Debug().Msgf("Server indicated session closed")
						errChan <- errors.New("session closed by server")

						return
					}
					if len(data) > 0 && string(data) != "EMPTY" {
						if _, err := conn.Write(data); err != nil {
							log.Debug().Msgf("Error writing to connection: %v", err)
							errChan <- err

							return
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
		log.Debug().Msgf("Session ended: %v", err)
	case <-done:
	}
}

// sendChunk sends a chunk of data through DNS.
func (c *DNSClient) sendChunk(chunk []byte, sequence uint16) error {
	// Split large chunks into smaller ones
	maxChunkSize := 100 // Reduced chunk size

	chunks := blind.SplitDataIntoChunks(chunk, maxChunkSize)

	for i, subChunk := range chunks {
		encodedData := blind.EncodeDNSSafe(subChunk)

		// Construct FQDN
		fqdn := fmt.Sprintf("%s.%04x.%s.%s",
			encodedData,
			//nolint:gosec
			sequence+uint16(i),
			c.sessionID,
			c.tld)

		_, err := c.sendQuery(fqdn)
		if err != nil {
			//nolint:gosec
			return fmt.Errorf("failed to send chunk %d: %w", sequence+uint16(i), err)
		}
	}

	return nil
}

// sendQuery sends a DNS query and returns the response.
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
		r, _, err := c.dnsClient.Exchange(msg, c.dnsServer)
		if err != nil {
			if strings.Contains(err.Error(), "i/o timeout") {
				log.Debug().Msgf("Query failed: %v, retrying...", err)
				time.Sleep(blind.RetryDelay)

				continue
			}

			return nil, err
		}

		if r.Rcode != dns.RcodeSuccess {
			log.Debug().Msgf("Query returned error code %d, retrying...", r.Rcode)
			time.Sleep(blind.RetryDelay)

			continue
		}

		if len(r.Answer) > 0 {
			if txt, ok := r.Answer[0].(*dns.TXT); ok {
				responseText := strings.Join(txt.Txt, "")
				log.Trace().Msgf("TXT response: %s", responseText)
				if responseText == "EMPTY" {
					return nil, nil
				}

				decodedResponse, err := blind.DecodeDNSSafe(responseText)
				if err != nil {
					log.Debug().Msgf("Failed to decode response: %v", err)

					return nil, err
				}
				if string(decodedResponse) == "EMPTY" {
					return nil, nil
				}

				return decodedResponse, nil
			}
		}

		return nil, nil
	}

	return nil, errors.New("max retries exceeded")
}

// pollForData polls the server for available data.
func (c *DNSClient) register() ([]byte, error) {
	fqdn := fmt.Sprintf("%s.aaaa.%s.%s", c.agentID, c.sessionID, c.tld)

	response, err := c.sendQuery(fqdn)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "EMPTY" {
		return nil, nil
	}

	return response, nil
}

// pollForData polls the server for available data.
func (c *DNSClient) pollForData() ([]byte, error) {
	fqdn := fmt.Sprintf("AA.ffff.%s.%s", c.sessionID, c.tld)

	response, err := c.sendQuery(fqdn)
	if err != nil {
		return nil, err
	}

	if len(response) == 0 || string(response) == "EMPTY" {
		return nil, nil
	}

	return response, nil
}

// sendData sends data through DNS.
func (c *DNSClient) sendData(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	// Start with sequence 0
	//nolint:gosec
	sequence := uint16(0)

	// Send data in chunks
	return c.sendChunk(data, sequence)
}
