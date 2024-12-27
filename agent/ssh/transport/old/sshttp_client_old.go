package old

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"Goauld/common/log"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"golang.org/x/time/rate"
	"io"
	"net"
	"net/http"
	"sync"
	"time"
)

const (
	maxBodySize    = 10 * 1024 * 1024
	readWriteSize  = 32 * 1024
	bufferPoolSize = 64 * 1024
)

type SSHTTPClient struct {
	destUrl         string
	httpClient      *http.Client
	rateLimiter     *rate.Limiter
	sessionId       string
	sessions        sync.Map
	readBufferSize  int
	writeBufferSize int
	pollInterval    time.Duration
	batchSize       int
	maxBodySize     int
	bufferPool      sync.Pool
}

type sessionInfo struct {
	ssh2httpBuffer bytes.Buffer
	http2sshBuffer bytes.Buffer
	lastActive     time.Time
	done           chan struct{}
	closeOnce      sync.Once
}

func (s *sessionInfo) close() {
	s.closeOnce.Do(func() {
		close(s.done)
	})
}

func sessionId() (string, error) {
	buf := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, buf)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}

func NewSSHTTPClient() (*SSHTTPClient, error) {

	sess, err := sessionId()
	if err != nil {
		return nil, fmt.Errorf("error generating session id %w", err)
	}

	transport := &http.Transport{
		MaxIdleConns:          1,
		IdleConnTimeout:       90 * time.Second,
		DisableCompression:    true,
		ForceAttemptHTTP2:     false,
		MaxIdleConnsPerHost:   1,
		MaxConnsPerHost:       1,
		WriteBufferSize:       32 * 1024,
		ReadBufferSize:        32 * 1024,
		ResponseHeaderTimeout: 30 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	transport = proxy.ProxifyTransport(transport)

	client := &SSHTTPClient{
		destUrl: agent.Get().SSHTTPUrl(),
		httpClient: &http.Client{
			Timeout:   time.Second * 30,
			Transport: transport,
		},
		rateLimiter:     rate.NewLimiter(rate.Every(time.Millisecond), 1000),
		sessionId:       sess,
		readBufferSize:  readWriteSize,
		writeBufferSize: readWriteSize,
		pollInterval:    50 * time.Millisecond,
		batchSize:       readWriteSize,
		maxBodySize:     maxBodySize,
		bufferPool: sync.Pool{New: func() interface{} {
			return make([]byte, bufferPoolSize)
		}},
	}
	return client, nil
}

func (c *SSHTTPClient) Request(method string, sshttpServerUrl string, body io.Reader, closeConnection bool) (*http.Request, error) {
	req, err := http.NewRequest(method, sshttpServerUrl, body)
	if err != nil {
		return nil, err
	}
	req.Host = req.URL.Host

	if closeConnection {
		req.Header.Set("X-Connection-Close", "true")
	}

	log.Trace().Msgf("request url: %s", req.URL)
	return req, nil
}

func (c *SSHTTPClient) HandleConnection() {
	ctx, cancel := context.WithTimeout(context.Background(), 24*time.Hour)
	defer cancel()

	sessionId := c.sessionId
	buffer := c.bufferPool.Get().([]byte)
	defer c.bufferPool.Put(buffer)

	session := &sessionInfo{
		ssh2httpBuffer: bytes.Buffer{},
		http2sshBuffer: bytes.Buffer{},
		lastActive:     time.Now(),
		done:           make(chan struct{}),
	}

	c.sessions.Store(sessionId, session)
	defer c.sessions.Delete(sessionId)
	defer session.close()

	go func() {
		ticker := time.NewTicker(c.pollInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-session.done:
				return
			case <-ticker.C:
				err := c.pollData(ctx, sessionId)
				if err != nil && !errors.Is(err, io.EOF) {
					log.Warn().Err(err).Msgf("error polling data %s", sessionId)

				}
			}
		}
	}()

	for {
		n, err := session.ssh2httpBuffer.Read(buffer) //c.session.Read(buffer)
		if err != nil {
			if err != io.EOF {
				log.Warn().Err(err).Msgf("error reading data %s", sessionId)
			}
			session.close()
			break
		}
		if n > 0 {
			data := make([]byte, n)
			copy(data, buffer[:n])
			err := c.sendData(ctx, sessionId, data, false)
			if err != nil {
				log.Warn().Err(err).Msgf("error sending data %s", sessionId)
				session.close()
				break
			}
		}
	}

	req, err := c.Request(http.MethodPost, c.destUrl, nil, true)
	if err == nil {
		req = req.WithContext(context.Background())
		req.Header.Set("X-For", sessionId)
		req.Header.Set("X-Connection-Close", "true")
		resp, err := c.httpClient.Do(req)
		if err == nil {
			resp.Body.Close()
		}

	}
}

func (c *SSHTTPClient) sendData(ctx context.Context, sessionId string, data []byte, closeConnection bool) error {
	log.Trace().Msgf("sending data to %s", sessionId)
	req, err := c.Request(http.MethodPost, c.destUrl, bytes.NewReader(data), closeConnection)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Set("X-For", sessionId)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	log.Trace().Msgf("got response from %s: ", sessionId, resp.StatusCode)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code %d", resp.StatusCode)
	}
	return nil
}

func (c *SSHTTPClient) handleResponseError(resp *http.Response, body []byte) {
	if resp.StatusCode != http.StatusOK {
		log.Error().Msgf("unexpected status code %d", resp.StatusCode)
		log.Debug().Msgf("body: %s", string(body))
	}
}

func (c *SSHTTPClient) pollData(ctx context.Context, sessionId string) error {
	s, ok := c.sessions.Load(sessionId)
	if !ok {
		return fmt.Errorf("session not found %s", sessionId)
	}
	session := s.(*sessionInfo)
	req, err := c.Request(http.MethodGet, c.destUrl, nil, true)
	if err != nil {
		return err
	}

	req = req.WithContext(ctx)
	req.Header.Set("X-For", sessionId)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.handleResponseError(resp, body)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBodySize))
	if err != nil {
		return err
	}

	if len(data) > 0 {
		if bytes.Contains(data, []byte("<!DOCTYPE html>")) || bytes.Contains(data, []byte("<html>")) {
			c.handleResponseError(resp, data)
		}

		decoded, err := hex.DecodeString(string(data))
		if err != nil {
			return fmt.Errorf("error decoding data %s: %w", decoded, err)
		}
		_, err = session.http2sshBuffer.Write(decoded) //conn.Write(decoded)
		if err != nil {
			return fmt.Errorf("error sending data to %s: %w", sessionId, err)
		}
	}
	return nil
}

func (c *SSHTTPClient) Read(b []byte) (int, error) {
	s, ok := c.sessions.Load(c.sessionId)
	if !ok {
		return 0, fmt.Errorf("session not found %s", c.sessionId)
	}
	session := s.(*sessionInfo)
	return session.ssh2httpBuffer.Read(b)
}

func (c *SSHTTPClient) Write(b []byte) (int, error) {
	s, ok := c.sessions.Load(c.sessionId)
	if !ok {
		return 0, fmt.Errorf("session not found %s", c.sessionId)
	}
	session := s.(*sessionInfo)
	return session.ssh2httpBuffer.Write(b)
}

func (c *SSHTTPClient) Close() error {
	return nil
}

func (c *SSHTTPClient) LocalAddr() net.Addr {
	return nil
}

func (c *SSHTTPClient) RemoteAddr() net.Addr {
	return nil
}

func (c *SSHTTPClient) SetDeadline(t time.Time) error {
	return nil
}

func (c *SSHTTPClient) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *SSHTTPClient) SetWriteDeadline(t time.Time) error {
	return nil
}
