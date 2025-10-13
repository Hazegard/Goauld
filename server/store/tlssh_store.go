package store

import (
	"Goauld/common/types"
	"errors"
	"net"
)

// TLSSHAgent handle the SSH over TLS store.
type TLSSHAgent struct {
	TLSConn net.Conn
	SSHConn net.Conn
}

// TLSSHAddAgent adds the TLS connections to the TLSSH agent store.
func (a *AgentStore) TLSSHAddAgent(id string, tlsshAgent *TLSSHAgent) {
	a.tlsshAgentMapMu.Lock()
	a.tlsshAgentMap[id] = tlsshAgent
	a.tlsshAgentMapMu.Unlock()
}

// TLSSHRemoveAgent removes the TLS connections of the TLSSH agent.
func (a *AgentStore) TLSSHRemoveAgent(id string) {
	a.tlsshAgentMapMu.Lock()
	delete(a.tlsshAgentMap, id)
	a.tlsshAgentMapMu.Unlock()
}

// TLSSHCloseAgent closes the TLS connections of the TLSSH agent.
func (a *AgentStore) TLSSHCloseAgent(id string) error {
	a.tlsshAgentMapMu.Lock()
	agent := a.tlsshAgentMap[id]
	a.tlsshAgentMapMu.Unlock()
	a.TLSSHRemoveAgent(id)
	var errs []error
	if agent != nil && agent.TLSConn != nil {
		err := agent.SSHConn.Close()
		errs = append(errs, err)
	}
	if agent != nil && agent.TLSConn != nil {
		err := agent.SSHConn.Close()
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// DumpTLSSH return the TLSSH information associated to the agent.
func (a *AgentStore) DumpTLSSH(id string) types.TLSSHState {
	a.tlsshAgentMapMu.Lock()
	agent := a.tlsshAgentMap[id]
	defer a.tlsshAgentMapMu.Unlock()
	state := types.TLSSHState{
		AgentID: id,
	}
	if agent == nil {
		return state
	}
	if agent.TLSConn != nil {
		state.TLSConn = types.Conn{
			LocaleAddr: agent.TLSConn.LocalAddr().String(),
			RemoteAddr: agent.TLSConn.RemoteAddr().String(),
		}
	}
	if agent.SSHConn != nil {
		state.SSHConn = types.Conn{
			LocaleAddr: agent.SSHConn.LocalAddr().String(),
			RemoteAddr: agent.SSHConn.RemoteAddr().String(),
		}
	}

	return state
}
