package transport

import (
	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/store"
	"errors"
	"io"
	"net"
)

type TLSSHServer struct {
	store *store.AgentStore
}

func NewTLSSHServer(store *store.AgentStore) *TLSSHServer {
	return &TLSSHServer{
		store: store,
	}
}

func (tlssh *TLSSHServer) HandleTLSSH(tlsConn net.Conn, id string) {

	sshConn, err := net.Dial("tcp", config.Get().LocalSShServer())
	if err != nil {
		log.Error().Err(err).Msg("error connecting to server")
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
			log.Error().Err(err).Msgf("TLSSH: TLS -> SSH connection failed (%s)", id)
			errChan <- err
		}
	}()
	go func() {
		_, err := io.Copy(sshConn, tlsConn)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Error().Err(err).Msgf("TLSSH: SSH -> TLS connection failed (%s)", id)
			errChan <- err
		}
	}()
	err = <-errChan
	if err != nil {
		log.Error().Err(err).Msgf("TLSSH: error during copy (%s)", id)
	}

	err = tlssh.store.TlsshCloseAgent(id)
	if err != nil {
		log.Error().Err(err).Msgf("WSSH: error while closing websocket streams (%s)", id)
	}
}
