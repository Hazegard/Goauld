package transport

import (
	"errors"
	"io"
	"net"

	"github.com/quic-go/quic-go"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
)

// QUICServer is the struct used to handle SSH over QUIC.
type QUICServer struct {
	agentStore *store.AgentStore
	db         *persistence.DB
}

// NewQUICServer returns a new QUICSHServer.
func NewQUICServer(agentStore *store.AgentStore, db *persistence.DB) *QUICServer {
	return &QUICServer{
		agentStore: agentStore,
		db:         db,
	}
}

// HandleQuic handle the SSH over QUIC connection
// It initializes a connection to the SSH server
// and start bidirectional communication between the QUIC connection
// and the SSH connection.
func (qssh *QUICServer) HandleQuic(quicConn *quic.Stream, id string, remote string) {
	// Initializes a connection to the SSH server
	sshConn, err := net.Dial("tcp", config.Get().LocalSSHAddr())
	if err != nil {
		log.Error().Str("ID", id).Str("Mode", "QUIC").Err(err).Msg("error connecting to server")

		return
	}
	// Adds the agent to the QUIC over SSH store
	QUICshAgent := &store.QUICAgent{QUICStream: quicConn, SSHConn: sshConn}
	qssh.agentStore.QuicAddAgent(id, QUICshAgent)

	errChan := make(chan error, 1)
	// Initializes the QUIC to SSH connection
	go func() {
		_, err := io.Copy(quicConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "QUIC").Err(err).Msg("QUIC -> SSH connection failed")
			errChan <- err
		}
	}()
	// Initializes the SSH to QUIC connection
	go func() {
		_, err := io.Copy(sshConn, quicConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "QUIC").Err(err).Msg("SSH -> QUIC connection failed")
			errChan <- err
		}
	}()
	// Adds the QUIC mode of the agent to the database
	err = qssh.db.SetAgentSSHMode(id, "QUIC", remote)
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "QUIC").Msg("error setting agent mode to QUIC")
	}
	// Waits for an error to occur, either in the
	// SSH -> QUIC connection or in the QUIC -> SSH connection
	err = <-errChan
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "QUIC").Msg("error during copy")
	}

	// Closes all remaining connections of the agent
	err = qssh.agentStore.TLSSHCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "QUIC").Msg("error while closing QUIC streams")
	}
	// Updates the database to set the agent mode as disconnected
	err = qssh.db.SetAgentSSHMode(id, "OFF", "")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "QUIC").Msg("error setting agent mode to OFF")
	}
}
