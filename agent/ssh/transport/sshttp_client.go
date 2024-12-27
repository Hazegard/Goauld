package transport

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"Goauld/common/log"
	"bytes"
	"errors"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

type SSHHttpClient struct {
	client *http.Client
	buffer bytes.Buffer
	bufMu  sync.Mutex
	url    string
}

func NewSSHTTPConn() *SSHHttpClient {
	httpClient := &http.Client{
		Transport: proxy.NewTransportProxy(),
	}
	return &SSHHttpClient{
		client: httpClient,
		url:    agent.Get().SSHTTPUrl(),
	}
}

func (c *SSHHttpClient) Connect() error {
	log.Trace().Msgf("[SSHTTP] Connect to %s", c.url)
	r, err := c.client.Head(c.url)

	if err != nil {
		return err
	}
	if r.StatusCode != 200 {
		return errors.New("HEAD: " + r.Status)
	}
	log.Trace().Msgf("[SSHTTP] HTTP successfully connected: %s", r.Status)
	return nil
}

func (c *SSHHttpClient) Read(b []byte) (int, error) {
	c.bufMu.Lock()
	defer c.bufMu.Unlock()
	//log.Trace().Msgf("READ START")
	//defer log.Trace().Msgf("READ END")
	if c.buffer.Len() > 0 {
		//log.Trace().Msgf("read data in buffer")
		return c.buffer.Read(b)
	}

	r, err := c.client.Get(c.url)
	if err != nil {
		return 0, err
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		//log.Trace().Msgf("read body err: %v", err)
		return 0, err
	}
	r.Body.Close()

	_, err = c.buffer.Write(body)
	if err != nil {
		//fmt.Printf("error writing to buffer %v\n", err)
	}
	return c.buffer.Read(b)

}

func (c *SSHHttpClient) Write(b []byte) (int, error) {
	//log.Trace().Msgf("WRITE START")
	//defer log.Trace().Msgf("WRITE END")
	_, err := c.client.Post(c.url, "application/octet-stream", bytes.NewReader(b))
	if err != nil {
		return 0, err
	}
	//log.Trace().Msgf("http client post response: %s", r.Status)
	return len(b), nil
}

func (c *SSHHttpClient) Close() error {
	log.Trace().Msgf("[SSHTTP] Close() called")
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
