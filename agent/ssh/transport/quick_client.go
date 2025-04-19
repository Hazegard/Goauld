package transport

import (
	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/quic-go/quic-go"
	"net"
)

type StreamConn struct {
	quic.Stream
	lAddr net.Addr
	rADdr net.Addr
}

func (s *StreamConn) LocalAddr() net.Addr  { return s.lAddr }
func (s *StreamConn) RemoteAddr() net.Addr { return s.rADdr }

func GetQuicConn(ctx context.Context) (*StreamConn, error) {

	tlsConf := proxy.NewTlsConfig()
	tlsConf.NextProtos = []string{"quic"}
	tlsConf.MinVersion = tls.VersionTLS13

	quicConf := &quic.Config{}
	conn, err := quic.DialAddr(ctx, config.Get().QuicUrl(), tlsConf, quicConf)
	if err != nil {
		return nil, fmt.Errorf("error dialing QUIC address: %s", err)
	}
	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		return nil, fmt.Errorf("error opening QUIC stream: %s", err)
	}

	// Write the agent ID as header to allow the server to identify which agent
	// is currently connecting
	_, err = stream.Write([]byte(config.Get().Id))
	if err != nil {
		return nil, err
	}
	sc := &StreamConn{
		Stream: stream,
		lAddr:  conn.LocalAddr(),
		rADdr:  conn.RemoteAddr(),
	}

	return sc, err
}
