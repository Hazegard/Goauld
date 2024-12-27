package test

import (
	"Goauld/common/log"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"
)

const (
	Establish = "establish"
	Forward   = "forward"
	Require   = "require"
	Read      = "read"
	Write     = "write"
	Goodbye   = "goodbye"
)

const BuffSize = 10000

type agent struct {
	conn   net.Conn
	id     string
	url    string
	target string
}

func newAgent(conn net.Conn, url, target string) *agent {
	return &agent{
		conn:   conn,
		id:     uuid.New().String()[0:8],
		url:    url,
		target: target,
	}
}

func (a *agent) work() {
	io.Pipe()
	log.Info().Str("id", a.id).
		Str("remote", a.conn.RemoteAddr().String()).
		Msgf("new connection established")

	var err error
	ctx := context.Background()

	defer func() {
		a.conn.Close()
		log.Info().
			Str("id", a.id).
			Str("client", a.conn.RemoteAddr().String()).
			Msg("connection closed")
	}()

	if _, err = a.proxy(ctx, Establish, nil); err != nil {
		log.Error().Err(err).Str("id", a.id).Msg("proxy failed")
		return
	}

	go a.write(ctx)

	buff := make([]byte, BuffSize)
	n := 0
	for {
		if n, err = a.conn.Read(buff); err == nil {
			_, _ = a.proxy(ctx, Write, buff[0:n])
		}

		if errors.Is(err, io.EOF) {
			_, _ = a.proxy(ctx, Goodbye, nil)
			_ = a.conn.Close()
			return
		}

		if err != nil {
			log.Error().Err(err).Str("id", a.id).Msg("proxy failed")
			return
		}
	}
}

func (a *agent) write(ctx context.Context) {
	for {
		raw, err := a.proxy(ctx, Read, nil)
		if err != nil {
			log.Error().Err(err).Str("id", a.id).Msg("proxy failed")
			break
		}

		if len(raw) > 0 {
			if _, err = a.conn.Write(raw); err != nil {
				log.Error().Err(err).Str("id", a.id).Msg("tcp write data failed")
				break
			}

			log.Debug().Str("id", a.id).Msgf("received %d bytes", len(raw))
		}
	}
}

func (a *agent) proxy(ctx context.Context, action string, data []byte) ([]byte, error) {
	ctxTimeout, cancel := context.WithTimeout(ctx, time.Minute*6)
	defer cancel()

	var reader io.Reader
	if len(data) > 0 {
		log.Debug().Str("id", a.id).Msgf("send %d bytes", len(data))

		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctxTimeout, http.MethodPost, a.url, reader)
	if err != nil {
		return nil, fmt.Errorf("new http request: %w", err)
	}

	req.Header.Set("Proxy-Id", a.id)
	req.Header.Set("Proxy-Target", a.target)
	req.Header.Set("Proxy-Action", action)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("do http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("do http request: status code %d", resp.StatusCode)
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}
	return raw, nil
}
