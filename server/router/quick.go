package router

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"context"
	"net"

	"github.com/quic-go/quic-go"
)

// ServeQUIC start a TLS listener on the configured port.
func (router *MainRouter) ServeQUIC() {
	httpsAddr := config.Get().LocalHTTPSAddr()
	quicConf := &quic.Config{
		EnableDatagrams: true,
	}
	listener, err := quic.ListenAddr(httpsAddr, router.server3.TLSConfig, quicConf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start QUIC listener")

		return
	}
	//nolint:errcheck
	defer listener.Close()
	//nolint:forcetypeassert
	config.Get().UpdateQUICAddr(listener.Addr().(*net.UDPAddr).Port)
	log.Info().Str("Address", config.Get().QuicAddr).Msgf("QUIC server listening")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to accept connection")

			continue
		}
		go router.HandleQUIC(conn)
	}
}

// HandleQUIC handle the incoming TLS request
// If the request matched the HTTP domain, forward this request to the HTTP router
// If the request matches the TLS domain, forward this TLS traffic to the SSH over TLS.
func (router *MainRouter) HandleQUIC(c *quic.Conn) {
	alpn := c.ConnectionState().TLS.NegotiatedProtocol

	switch alpn {
	case "h3":
		err := router.server3.ServeQUICConn(c)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to serve QUIC connection")

			return
		}
	case "quic":
		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("Failed to accept stream")

			return
		}
		// The client first sends its ID before transferring the conn to the SSH client
		// The ID is a MD5 hash
		rawID := make([]byte, 32)
		n, err := stream.Read(rawID)
		if err != nil {
			log.Error().Err(err).Msg("QUIC read ID fail")

			return
		}
		id := string(rawID[:n])
		log.Info().Str("ID", id).Msg("Receiving incoming SSH connection over QUIC")

		router.quicSSH.HandleQuic(stream, id, c.RemoteAddr().String())
	}
}
