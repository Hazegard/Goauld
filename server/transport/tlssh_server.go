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
	agentStore *store.AgentStore
	db         *persistence.DB
}

// NewTLSSHServer returns a new TLSSHServer.
func NewTLSSHServer(agentStore *store.AgentStore, db *persistence.DB) *TLSSHServer {
	return &TLSSHServer{
		agentStore: agentStore,
		db:         db,
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

	if id == "00000000000000000000000000000000" {
		HandleHealthCheckTls(tlsConn, sshConn)
		return
	}
	// Adds the agent to the TLS over SSH store
	tlsshAgent := &store.TLSSHAgent{TLSConn: tlsConn, SSHConn: sshConn}
	tlssh.agentStore.TLSSHAddAgent(id, tlsshAgent)

	d1 := make(chan struct{})
	d2 := make(chan struct{})
	// Initializes the TLS to SSH connection
	go func() {
		_, err := io.Copy(tlsConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("TLS -> SSH connection failed")
			d1 <- struct{}{}
		}
	}()
	// Initializes the SSH to TLS connection
	go func() {
		_, err := io.Copy(sshConn, tlsConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "TLS").Err(err).Msg("SSH -> TLS connection failed")
			d2 <- struct{}{}
		}
	}()
	// Adds the TLS mode of the agent to the database
	err = tlssh.db.SetAgentSSHMode(id, "TLS", tlsConn.RemoteAddr().String())
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error setting agent mode to TLS")
	}
	// Waits for an error to occur, either in the
	// SSH -> TLS connection or in the TLS -> SSH connection

	select {
	case <-d1:
	case <-d2:
	}
	// Closes all remaining connections of the agent
	err = tlssh.agentStore.TLSSHCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error while closing TLS streams")
	}
	// Updates the database to set the agent mode as disconnected
	err = tlssh.db.SetAgentSSHMode(id, "OFF", "")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "TLS").Msg("error setting agent mode to OFF")
	}
}

func HandleHealthCheckTls(tlsConn net.Conn, sshConn net.Conn) {
	go func() {
		_, _ = io.Copy(tlsConn, sshConn)
	}()
	defer func() {
		_ = tlsConn.Close()
		_ = sshConn.Close()
	}()

	_, _ = io.Copy(sshConn, tlsConn)
}
