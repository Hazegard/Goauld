package store

import (
	"errors"
	"net"
	"strings"

	"Goauld/common/types"
)

type TLSSHAgent struct {
	TLSConn net.Conn
	SSHConn net.Conn
}

// TlsshAddAgent adds the TLS connections to the TLSSH agent store
func (a *AgentStore) TlsshAddAgent(id string, tlsshAgent *TLSSHAgent) {
	a.tlsshAgentMapMu.Lock()
	a.tlsshAgentMap[id] = tlsshAgent
	a.tlsshAgentMapMu.Unlock()
}

// TlsshRemoveAgent removes the TLS connections of the TLSSH agent
func (a *AgentStore) TlsshRemoveAgent(id string) {
	a.tlsshAgentMapMu.Lock()
	delete(a.tlsshAgentMap, id)
	a.tlsshAgentMapMu.Unlock()
}

// TlsshCloseAgent closes the TLS connections of the TLSSH agent
func (a *AgentStore) TlsshCloseAgent(id string) error {
	a.tlsshAgentMapMu.Lock()
	agent := a.tlsshAgentMap[id]
	a.tlsshAgentMapMu.Unlock()
	a.TlsshRemoveAgent(id)
	var errs []string
	if agent != nil && agent.TLSConn != nil {
		err1 := agent.SSHConn.Close()
		if err1 != nil {
			errs = append(errs, err1.Error())
		}
	}
	if agent != nil && agent.TLSConn != nil {
		err2 := agent.SSHConn.Close()
		if err2 != nil {
			errs = append(errs, err2.Error())
		}
	}
	return errors.New(strings.Join(errs, " / "))
}

// DumpTLSSH return the TLSSH information associated to the agent
func (a *AgentStore) DumpTLSSH(id string) types.TLSSHState {
	a.tlsshAgentMapMu.Lock()
	agent := a.tlsshAgentMap[id]
	defer a.tlsshAgentMapMu.Unlock()
	state := types.TLSSHState{
		AgentId:       id,
		SSHLocaleAddr: agent.SSHConn.LocalAddr().String(),
		SSHRemoteAddr: agent.SSHConn.RemoteAddr().String(),
		TLSLocaleAddr: agent.TLSConn.LocalAddr().String(),
		TLSRemoteAddr: agent.TLSConn.RemoteAddr().String(),
	}

	return state
}
