package http

import (
	"Goauld/agent/proxy"
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/xtaci/kcp-go/v5"
	"github.com/xtaci/smux"
	"io"
	"log"
	"net/http"
	"time"
	"www.bamsoftware.com/git/champa.git/turbotunnel"
)

type SSHTTP struct {
	Session *smux.Session
	Pconn   *PollingPacketConn
	KcpConn *kcp.UDPSession
	Stream  *smux.Stream
	Client  *http.Client
}

func (s *SSHTTP) Close() error {
	var errs []error
	errs = append(errs, s.Stream.Close())
	errs = append(errs, s.Pconn.Close())
	errs = append(errs, s.Session.Close())
	errs = append(errs, s.KcpConn.Close())
	return errors.Join(errs...)
}

const (
	idleTimeout = 2 * time.Minute
)

func NewSSHTTP(serverURL string) (*SSHTTP, error) {

	// http.DefaultTransport.(*http.Transport).MaxConnsPerHost = 20

	httpClient := proxy.NewHttpClientProxy()
	httpClient.Transport.(*http.Transport).MaxConnsPerHost = 20

	var poller PollFunc = func(ctx context.Context, client *http.Client, p []byte) (io.ReadCloser, error) {
		return poll(ctx, httpClient, serverURL, p)
	}
	pconn := NewPollingPacketConn(turbotunnel.DummyAddr{}, poller, httpClient)

	// Open a KCP conn over the Noise layer.
	conn, err := kcp.NewConn2(turbotunnel.DummyAddr{}, nil, 0, 0, pconn)
	if err != nil {
		return nil, fmt.Errorf("opening KCP conn: %v", err)
	}

	log.Printf("begin session %08x", conn.GetConv())
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
	// ACK received data immediately; this is good in our polling model.
	conn.SetACKNoDelay(true)
	conn.SetWindowSize(1024, 1024) // Default is 32, 32.
	// TODO: We could optimize a call to conn.SetMtu here, based on a
	// maximum URL length we want to send (such as the 8000 bytes
	// recommended at https://datatracker.ietf.org/doc/html/rfc7230#section-3.1.1).
	// The idea is that if we can slightly reduce the MTU from its default
	// to permit one more packet per request, we should do it.
	// E.g. 1400*5 = 7000, but 1320*6 = 7920.

	// Start a smux session on the Noise channel.
	smuxConfig := smux.DefaultConfig()
	smuxConfig.Version = 2
	smuxConfig.KeepAliveTimeout = idleTimeout
	smuxConfig.MaxReceiveBuffer = 4 * 1024 * 1024 // default is 4 * 1024 * 1024
	smuxConfig.MaxStreamBuffer = 1 * 1024 * 1024  // default is 65536
	sess, err := smux.Client(conn, smuxConfig)
	if err != nil {
		return nil, fmt.Errorf("opening smux session: %v", err)
	}

	stream, err := sess.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("opening stream: %v", err)
	}

	return &SSHTTP{
		Session: sess,
		Pconn:   pconn,
		KcpConn: conn,
		Stream:  stream,
		Client:  httpClient,
	}, nil
}

func poll(ctx context.Context, httpClient *http.Client, serverURL string, p []byte) (io.ReadCloser, error) {
	// Append a cache buster and the encoded p to the path of serverURL.

	req, err := http.NewRequest("POST", serverURL, bytes.NewReader(p))
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)
	req.Header.Set("User-Agent", "") // Disable default "Go-http-client/1.1".

	// TODO: use proxy here

	resp, err := httpClient.Transport.RoundTrip(req) //http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("server returned status %v", resp.Status)
	}

	a, err := io.ReadAll(resp.Body)
	if err != nil {
		resp.Body.Close()
		fmt.Println(err)
	}

	// The caller should read from the decoder (which reads from the
	// response body), but close the actual response body when done.
	return &struct {
		io.Reader
		io.Closer
	}{
		Reader: bytes.NewReader(a),
		Closer: resp.Body,
	}, nil
}
