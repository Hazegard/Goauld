package transport

import (
	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"context"
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

func GetQuickConn(ctx context.Context) (*StreamConn, error) {

	tlsConf := proxy.NewTlsConfig()
	tlsConf.NextProtos = []string{"ssh"}
	quicConf := &quic.Config{}
	conn, err := quic.DialAddr(ctx, config.Get().ServerUrl(), tlsConf, quicConf)
	if err != nil {
		return nil, err
	}
	stream, err := conn.OpenStreamSync(ctx)
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
