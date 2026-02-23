package router

import (
	"Goauld/common/utils"
	"crypto/tls"
	"net"

	"Goauld/common/log"
	"Goauld/server/config"
)

// ServeTLS start a TLS listener on the configured port.
func (router *MainRouter) ServeTLS() {
	httpsAddr := config.Get().LocalHTTPSAddr()
	listener, err := tls.Listen("tcp", httpsAddr, router.tlsConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start TLS listener")

		return
	}
	//nolint:errcheck
	defer listener.Close()
	//nolint:forcetypeassert
	config.Get().UpdateHTTPSAddr(listener.Addr().(*net.TCPAddr).Port)
	log.Info().Str("Address", config.Get().LocalHTTPSAddr()).Msgf("HTTPS server listening")

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Warn().Err(err).Msg("Failed to accept connection")

			continue
		}
		go router.HandleTLS(conn)
	}
}

// HandleTLS handle the incoming TLS request
// If the request matched the HTTP domain, forward this request to the HTTP router
// If the request matches the TLS domain, forward this TLS traffic to the SSH over TLS.
func (router *MainRouter) HandleTLS(c net.Conn) {
	//nolint:errcheck
	defer c.Close()
	// Check if the connection is a TLS connection

	if tlsConn, ok := c.(*tls.Conn); ok {
		// If the connection is TLS, performs the TLS handshake
		if err := tlsConn.Handshake(); err != nil {
			log.Warn().Err(err).Msg("TLS handshake failed")

			return
		}
		state := tlsConn.ConnectionState()
		// Extract the TLS SNI
		if utils.Contains(config.Get().HTTPDomain, state.ServerName) {
			// If the domain matches, the HTTP domain configured, serve the
			// request using the HTTP router

			err := router.server.Serve(NewSingleConnListener(tlsConn))
			if err != nil {
				log.Error().Err(err).Msg("Failed to start server")
			}
		} else if utils.Contains(config.Get().TLSDomain, state.ServerName) {
			// If the domain matches the TLS domain configured,
			// Handle it as a raw TLS

			// The client first sends its ID before transferring the conn to the SSH client
			// The ID is a MD5 hash
			rawID := make([]byte, 32)
			n, err := tlsConn.Read(rawID)
			if err != nil {
				log.Error().Err(err).Msg("TLS read ID fail")

				return
			}
			id := string(rawID[:n])
			if id != "00000000000000000000000000000000" {
				log.Info().Str("ID", id).Msg("Receiving incoming SSH connection over TLS")
			}
			// Serve the raw TLS traffic
			router.tlsshHandler.HandleTLSSH(c, id)
		}
	} else {
		log.Info().Msg("Invalid subdomain TLS connection received")
	}
}
