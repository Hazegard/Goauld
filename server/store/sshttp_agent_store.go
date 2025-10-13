package store

import (
	"errors"
	"net"

	"github.com/xtaci/smux"

	"Goauld/common/types"
)

// SSHTTPAgent handle the SSH over HTTP store.
type SSHTTPAgent struct {
	SSHConn net.Conn
	Stream  *smux.Stream
}

// SSHTTPAddAgent adds the ssh connection of agent id to the agent store.
func (a *AgentStore) SSHTTPAddAgent(ssh net.Conn, stream *smux.Stream, id string) {
	a.sshttpAgentMapMu.Lock()
	a.sshttpAgentMap[id] = &SSHTTPAgent{
		SSHConn: ssh,
		Stream:  stream,
	}
	a.sshttpAgentMapMu.Unlock()
}

// SSHTTPRemoveAgent removes the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHTTPRemoveAgent(id string) {
	a.sshttpAgentMapMu.Lock()
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
}

// SSHTTPGetAgent returns the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHTTPGetAgent(id string) *SSHTTPAgent {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	a.sshttpAgentMapMu.Unlock()

	return agent
}

// SSHTTPCloseAgent closes the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHTTPCloseAgent(id string) error {
	var err error
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
	if agent != nil && agent.SSHConn != nil {
		err = errors.Join(agent.SSHConn.Close(), agent.Stream.Close())
	}
	a.SSHTTPRemoveAgent(id)

	return err
}

// DumpSSHTTP return the SSHTTP information associated to the agent.
func (a *AgentStore) DumpSSHTTP(id string) types.SSHTTPState {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	defer a.sshttpAgentMapMu.Unlock()
	if agent == nil {
		return types.SSHTTPState{
			AgentID: id,
		}
	}
	state := types.SSHTTPState{
		AgentID: id,
		SSHConn: types.Conn{
			LocaleAddr: agent.SSHConn.LocalAddr().String(),
			RemoteAddr: agent.SSHConn.RemoteAddr().String(),
		},
		StreamConn: types.Conn{
			LocaleAddr: agent.Stream.LocalAddr().String(),
			RemoteAddr: agent.Stream.RemoteAddr().String(),
		},
		StreamID: agent.Stream.ID(),
	}

	return state
}
