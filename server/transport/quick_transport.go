package transport

import (
	"errors"
	"github.com/quic-go/quic-go"
	"io"
	"net"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
)

// QUICKServer is the struct used to handle SSH over QUICK
type QUICKServer struct {
	store *store.AgentStore
	db    *persistence.DB
}

// NewQUICKSHServer returns a new QUICKSHServer
func NewQUICKServer(store *store.AgentStore, db *persistence.DB) *QUICKServer {
	return &QUICKServer{
		store: store,
		db:    db,
	}
}

// HandleQUICKSH handle the SSH over QUICK connection
// It initializes a connection to the SSH server
// and start bidirectional communication between the QUICK connection
// and the SSH connection
func (qssh *QUICKServer) HandleQuick(quickConn quic.Stream, id string) {
	// Initializes a connection to the SSH server
	sshConn, err := net.Dial("tcp", config.Get().LocalSShAddr())
	if err != nil {
		log.Error().Str("ID", id).Str("Mode", "QUICK").Err(err).Msg("error connecting to server")
		return
	}
	// Adds the agent to the QUICK over SSH store
	QUICKshAgent := &store.QUICAgent{QUICStream: quickConn, SSHConn: sshConn}
	qssh.store.QuicAddAgent(id, QUICKshAgent)

	errChan := make(chan error, 1)
	// Initializes the QUICK to SSH connection
	go func() {
		_, err := io.Copy(quickConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "QUICK").Err(err).Msg("QUICK -> SSH connection failed")
			errChan <- err
		}
	}()
	// Initializes the SSH to QUICK connection
	go func() {
		_, err := io.Copy(sshConn, quickConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("Mode", "QUICK").Err(err).Msg("SSH -> QUICK connection failed")
			errChan <- err
		}
	}()
	// Adds the QUICK mode of the agent to the database
	err = qssh.db.SetAgentSshMode(id, "QUICK")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "QUICK").Msg("error setting agent mode to QUICK")
	}
	// Waits for an error to occur, either in the
	// SSH -> QUICK connection or in the QUICK -> SSH connection
	err = <-errChan
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "QUICK").Msg("error during copy")
	}

	// Closes all remaining connections of the agent
	err = qssh.store.TlsshCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("Mode", "QUICK").Msg("error while closing QUICK streams")
	}
	// Updates the database to set the agent mode as disconnected
	err = qssh.db.SetAgentSshMode(id, "OFF")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("Mode", "QUICK").Msg("error setting agent mode to OFF")
	}
}
