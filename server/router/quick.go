package router

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"context"
	"github.com/quic-go/quic-go"
	"net"
)

// ServeQuick start a TLS listener on the configured port
func (router *MainRouter) ServeQuick() {
	httpsAddr := config.Get().LocalHttpsAddr()
	quicConf := &quic.Config{
		EnableDatagrams: true,
	}
	listener, err := quic.ListenAddr(httpsAddr, router.tlsConfig, quicConf)
	if err != nil {
		log.Error().Err(err).Msg("Failed to start TLS listener")
		return
	}
	defer listener.Close()
	config.Get().UpdateHTTPSAddr(listener.Addr().(*net.TCPAddr).Port)
	log.Info().Str("Address", config.Get().LocalHttpsAddr()).Msgf("HTTPS server listening")

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Warn().Err(err).Msg("Failed to accept connection")
			continue
		}
		go router.HandleQuick(conn)
	}
}

// HandleQuick handle the incoming TLS request
// If the request matched the HTTP domain, forward this request to the HTTP router
// If the request matches the TLS domain, forward this TLS traffic to the SSH over TLS
func (router *MainRouter) HandleQuick(c quic.Connection) {

	alpn := c.ConnectionState().TLS.NegotiatedProtocol

	switch alpn {
	case "h3":
		router.server3.ServeQUICConn(c)
	case "ssh":
		stream, err := c.AcceptStream(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("Failed to accept stream")
			return
		}
		// The client first sends its ID before transferring the conn to the SSH client
		// The ID is a MD5 hash
		rawId := make([]byte, 128)
		n, err := stream.Read(rawId)
		if err != nil {
			log.Error().Err(err).Msg("TLS read ID fail")
			return
		}
		id := string(rawId[:n])
		log.Info().Str("ID", id).Msg("Receiving incoming SSH connection over TLS")

		router.quickSSH.HandleQuick(stream, id)
	}
}

/*func (router *MainRouter) quickssh(quickConn quic.Stream) {

	sshConn, err := net.Dial("tcp", config.Get().SshdAddr)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to SSH server")
		return
	}

	errChan := make(chan error, 1)
	// Initializes the QUIC to SSH connection
	go func() {
		_, err := io.Copy(quickConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("TLS -> SSH connection failed")
			errChan <- err
		}
	}()
	// Initializes the SSH to QUIC connection
	go func() {
		_, err := io.Copy(sshConn, quickConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("SSH -> TLS connection failed")
			errChan <- err
		}
	}()
	// Adds the QUIC mode of the agent to the database
	err = tlssh.db.SetAgentSshMode(id, "QUIC")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error setting agent mode to TLS")
	}
	// Waits for an error to occur, either in the
	// SSH -> TLS connection or in the TLS -> SSH connection
	err = <-errChan
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error during copy")
	}

	// Closes all remaining connections of the agent
	err = tlssh.store.TlsshCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error while closing TLS streams")
	}
	// Updates the database to set the agent mode as disconnected
	err = tlssh.db.SetAgentSshMode(id, "OFF")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error setting agent mode to OFF")
	}
}
*/
