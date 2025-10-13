package store

import (
	"Goauld/common/types"
	"errors"
	"net"
)

// WSSHAgent handle the SSH over Websocket store.
type WSSHAgent struct {
	srcConn net.Conn
	dstConn net.Conn
}

// WSSHAddAgent adds the WSSH connections to the store.
func (a *AgentStore) WSSHAddAgent(id string, src net.Conn, dst net.Conn) {
	a.wsshAgentMapMu.Lock()
	a.wsshAgentMap[id] = &WSSHAgent{srcConn: src, dstConn: dst}
	a.wsshAgentMapMu.Unlock()
}

// WSSHCloseAgent closes the WSSH connections.
func (a *AgentStore) WSSHCloseAgent(id string) error {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	a.wsshAgentMapMu.Unlock()
	a.WSSHRemoveAgent(id)
	var errs []error
	if agent != nil && agent.srcConn != nil {
		err1 := agent.srcConn.Close()
		if err1 != nil {
			errs = append(errs, err1)
		}
	}
	if agent != nil && agent.dstConn != nil {
		err2 := agent.dstConn.Close()
		if err2 != nil {
			errs = append(errs, err2)
		}
	}

	return errors.Join(errs...)
}

// WSSHRemoveAgent remove the agent of the store.
func (a *AgentStore) WSSHRemoveAgent(id string) {
	a.wsshAgentMapMu.Lock()
	delete(a.wsshAgentMap, id)
	a.wsshAgentMapMu.Unlock()
}

// WSSHGetAgent returns the WSSH connections of the agent.
func (a *AgentStore) WSSHGetAgent(id string) *WSSHAgent {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	a.wsshAgentMapMu.Unlock()

	return agent
}

// DumpWSSH return the WSSH information associated to the agent.
func (a *AgentStore) DumpWSSH(id string) types.WSSHState {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	defer a.wsshAgentMapMu.Unlock()
	if agent == nil {
		return types.WSSHState{
			AgentID: id,
		}
	}
	state := types.WSSHState{
		AgentID: id,
		SSHConn: types.Conn{
			LocaleAddr: agent.dstConn.LocalAddr().String(),
			RemoteAddr: agent.dstConn.RemoteAddr().String(),
		},
		WSConn: types.Conn{
			LocaleAddr: agent.srcConn.LocalAddr().String(),
			RemoteAddr: agent.srcConn.RemoteAddr().String(),
		},
	}

	return state
}
