package transport

import (
	"errors"
	"io"
	"net"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
)

// TLSSHServer is the struct used to handle SSH over TLS.
type TLSSHServer struct {
	store *store.AgentStore
	db    *persistence.DB
}

// NewTLSSHServer returns a new TLSSHServer.
func NewTLSSHServer(store *store.AgentStore, db *persistence.DB) *TLSSHServer {
	return &TLSSHServer{
		store: store,
		db:    db,
	}
}

// HandleTLSSH handle the SSH over TLS connection
// It initializes a connection to the SSH server
// and start bidirectional communication between the TLS connection
// and the SSH connection.
func (tlssh *TLSSHServer) HandleTLSSH(tlsConn net.Conn, id string) {
	// Initializes a connection to the SSH server
	sshConn, err := net.Dial("tcp", config.Get().LocalSSHAddr())
	if err != nil {
		log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("error connecting to server")

		return
	}
	// Adds the agent to the TLS over SSH store
	tlsshAgent := &store.TLSSHAgent{TLSConn: tlsConn, SSHConn: sshConn}
	tlssh.store.TLSSHAddAgent(id, tlsshAgent)

	errChan := make(chan error, 1)
	// Initializes the TLS to SSH connection
	go func() {
		_, err := io.Copy(tlsConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("TLS -> SSH connection failed")
			errChan <- err
		}
	}()
	// Initializes the SSH to TLS connection
	go func() {
		_, err := io.Copy(sshConn, tlsConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("SSH -> TLS connection failed")
			errChan <- err
		}
	}()
	// Adds the TLS mode of the agent to the database
	err = tlssh.db.SetAgentSSHMode(id, "TLS", tlsConn.RemoteAddr().String())
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
	err = tlssh.store.TLSSHCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error while closing TLS streams")
	}
	// Updates the database to set the agent mode as disconnected
	err = tlssh.db.SetAgentSSHMode(id, "OFF", "")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error setting agent mode to OFF")
	}
}
