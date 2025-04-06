package store

import (
	"errors"
	"github.com/xtaci/smux"
	"net"

	"Goauld/common/types"
)

type SSHTTPAgent struct {
	SshConn net.Conn
	Stream  *smux.Stream
}

// SshttpAddAgent adds the ssh connection of agent id to the agent store
func (a *AgentStore) SshttpAddAgent(ssh net.Conn, stream *smux.Stream, id string) {
	a.sshttpAgentMapMu.Lock()
	a.sshttpAgentMap[id] = &SSHTTPAgent{
		SshConn: ssh,
		Stream:  stream,
	}
	a.sshttpAgentMapMu.Unlock()
}

// SshttpRemoveAgent removes the ssh connection of the agent id to the agent store
func (a *AgentStore) SshttpRemoveAgent(id string) {
	a.sshttpAgentMapMu.Lock()
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
}

// SshttpGetAgent returns the ssh connection of the agent id to the agent store
func (a *AgentStore) SshttpGetAgent(id string) *SSHTTPAgent {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	a.sshttpAgentMapMu.Unlock()
	return agent
}

// SshttpCloseAgent closes the ssh connection of the agent id to the agent store
func (a *AgentStore) SshttpCloseAgent(id string) error {
	var err error
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
	if agent != nil && agent.SshConn != nil {
		errors.Join(agent.SshConn.Close(), agent.Stream.Close())
	}
	a.SshttpRemoveAgent(id)
	return err
}

// DumpSSHTTP return the SSHTTP information associated to the agent
func (a *AgentStore) DumpSSHTTP(id string) types.SSHTTState {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	defer a.sshttpAgentMapMu.Unlock()
	if agent == nil {
		return types.SSHTTState{
			AgentId: id,
		}
	}
	state := types.SSHTTState{
		AgentId: id,
		SshConn: types.Conn{
			LocaleAddr: agent.SshConn.LocalAddr().String(),
			RemoteAddr: agent.SshConn.RemoteAddr().String(),
		},
		StreamConn: types.Conn{
			LocaleAddr: agent.Stream.LocalAddr().String(),
			RemoteAddr: agent.Stream.RemoteAddr().String(),
		},
		StreamId: agent.Stream.ID(),
	}
	return state
}
