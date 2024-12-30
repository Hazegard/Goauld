package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"
	"errors"
	"io"
	"net"
)

type TLSSHServer struct {
	store *store.AgentStore
	db    *persistence.DB
}

func NewTLSSHServer(store *store.AgentStore, db *persistence.DB) *TLSSHServer {
	return &TLSSHServer{
		store: store,
		db:    db,
	}
}

func (tlssh *TLSSHServer) HandleTLSSH(tlsConn net.Conn, id string) {

	sshConn, err := net.Dial("tcp", config.Get().LocalSShServer())
	if err != nil {
		log.Error().Str("ID", id).Str("SSH Mode", "TLS").Err(err).Msg("error connecting to server")
		return
	}
	// go io.Copy(tlsConn, sshConn)
	// io.Copy(sshConn, tlsConn)
	tlsshAgent := &store.TLSSHAgent{TLSConn: tlsConn, SSHConn: sshConn}
	tlssh.store.TlsshAddAgent(id, tlsshAgent)

	errChan := make(chan error, 1)
	go func() {
		_, err := io.Copy(tlsConn, sshConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("SSH Mode", "TLS").Err(err).Msg("TLS -> SSH connection failed")
			errChan <- err
		}
	}()
	go func() {
		_, err := io.Copy(sshConn, tlsConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Str("ID", id).Str("SSH Mode", "TLS").Err(err).Msg("SSH -> TLS connection failed")
			errChan <- err
		}
	}()
	err = tlssh.db.SetAgentSshMode(id, "TLS")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("SSH Mode", "TLS").Msg("error setting agent mode to TLS")
	}
	err = <-errChan
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("SSH Mode", "TLS").Msg("error during copy")
	}

	err = tlssh.store.TlsshCloseAgent(id)
	if err != nil {
		log.Error().Str("ID", id).Err(err).Str("SSH Mode", "TLS").Msg("error while closing TLS streams")
	}
	err = tlssh.db.SetAgentSshMode(id, "DISCONNECTED")
	if err != nil {
		log.Warn().Str("ID", id).Err(err).Str("SSH Mode", "TLS").Msg("error setting agent mode to DISCONNECTED")
	}
}
