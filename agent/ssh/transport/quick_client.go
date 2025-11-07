package transport

import (
	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/quic-go/quic-go"
)

// StreamConn holds the QUIC information of a SSH over QUIC connection.
type StreamConn struct {
	*quic.Stream

	lAddr net.Addr
	rAddr net.Addr
}

// LocalAddr returns the local addr.
func (s *StreamConn) LocalAddr() net.Addr { return s.lAddr }

// RemoteAddr returns the remote addr.
func (s *StreamConn) RemoteAddr() net.Addr { return s.rAddr }

// GetQuicConn dials a QUIC connection and opens a stream, all respecting ctx.
func GetQuicConn(ctx context.Context, id string) (*StreamConn, error) {
	// 1) Prepare TLS and QUIC configs
	tlsConf := proxy.NewTLSConfig()
	tlsConf.NextProtos = []string{"quic"}
	tlsConf.MinVersion = tls.VersionTLS13

	quicConf := &quic.Config{}

	// 2) Dial QUIC with context
	conn, err := quic.DialAddr(ctx, config.Get().QuicURL(), tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("error dialing QUIC address: %w", err)
	}

	// 3) Open the stream with context
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		// Clean up the QUIC connection on failure
		_ = conn.CloseWithError(0, "stream open failed")

		return nil, fmt.Errorf("error opening QUIC stream: %w", err)
	}

	// 4) Write the agent ID header, bounded by ctx’s deadline (if any)
	if dl, ok := ctx.Deadline(); ok {
		// quic-go streams implement SetWriteDeadline
		_ = stream.SetWriteDeadline(dl)
	}
	header := []byte(id)
	if _, err := stream.Write(header); err != nil {
		_ = conn.CloseWithError(0, "header write failed")

		return nil, err
	}
	// Clear deadline so subsequent I/O isn’t affected
	_ = stream.SetWriteDeadline(time.Time{})

	// 5) Return a wrapper around the stream+addresses
	sc := &StreamConn{
		Stream: stream,
		lAddr:  conn.LocalAddr(),
		rAddr:  conn.RemoteAddr(),
	}

	return sc, nil
}
