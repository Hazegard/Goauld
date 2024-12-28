package router

import (
	"Goauld/common/log"
	"crypto/tls"
	"net"
)

func (router *HttpRouter) ServeTLS() {
	listener, err := tls.Listen("tcp", ":443", router.tlsConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start TLS listener")
		return
	}
	defer listener.Close()

	log.Info().Str("Address", ":443").Msgf("HTTPS server listening")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to accept connection")
			continue
		}
		go router.HandleTls(conn)
	}
}

func (router *HttpRouter) HandleTls(c net.Conn) {

	defer c.Close()
	// Check if connection is TLS
	if tlsConn, ok := c.(*tls.Conn); ok {
		if err := tlsConn.Handshake(); err != nil {
			log.Warn().Err(err).Msg("TLS handshake failed")
			return
		}
		state := tlsConn.ConnectionState()
		if state.ServerName == "a.hazegard.fr" {
			// Serve HTTPS traffic

			err := router.server.Serve(NewSingleConnListener(tlsConn))
			if err != nil {
				log.Error().Err(err).Msg("Failed to start server")
			}
		} else if state.ServerName == "b.hazegard.fr" {
			// The client first send its ID before transferring the conn to the SSH client
			// The ID is a MD5 hash
			rawId := make([]byte, 128)
			n, err := tlsConn.Read(rawId)
			if err != nil {
				log.Error().Err(err).Msg("TLS read ID fail")
				return
			}
			id := string(rawId[:n])
			log.Info().Str("ID", id).Msg("Receiving incomming SSH connection over TLS")
			// Serve RAW TLS traffic
			router.tlsshHandler.HandleTLSSH(c, id)
		}
	} else {
		log.Info().Msg("Non-TLS connection received")
	}

}
