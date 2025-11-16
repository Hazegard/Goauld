package store

import (
	"errors"
	"net"

	"Goauld/common/types"
)

// SSHCDNAgent handle the SSH over HTTP store.
type SSHCDNAgent struct {
	SSHConn net.Conn
}

// SSHCDNAddAgent adds the ssh connection of agent id to the agent store.
func (a *AgentStore) SSHCDNAddAgent(ssh net.Conn, id string) {
	a.sshCDNAgentMapMu.Lock()
	a.sshCDNAgentMap[id] = &SSHCDNAgent{
		SSHConn: ssh,
	}
	a.sshCDNAgentMapMu.Unlock()
}

// SSHCDNRemoveAgent removes the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHCDNRemoveAgent(id string) {
	a.sshCDNAgentMapMu.Lock()
	delete(a.sshCDNAgentMap, id)
	a.sshCDNAgentMapMu.Unlock()
}

// SSHCDNGetAgent returns the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHCDNGetAgent(id string) *SSHCDNAgent {
	a.sshCDNAgentMapMu.Lock()
	agent := a.sshCDNAgentMap[id]
	a.sshCDNAgentMapMu.Unlock()

	return agent
}

// SSHCDNCloseAgent closes the ssh connection of the agent id to the agent store.
func (a *AgentStore) SSHCDNCloseAgent(id string) error {
	var err error
	a.sshCDNAgentMapMu.Lock()
	agent := a.sshCDNAgentMap[id]
	delete(a.sshCDNAgentMap, id)
	a.sshCDNAgentMapMu.Unlock()
	if agent != nil && agent.SSHConn != nil {
		err = errors.Join(agent.SSHConn.Close())
	}
	a.SSHCDNRemoveAgent(id)

	return err
}

// DumpSSHCDN return the SSHCDN information associated to the agent.
func (a *AgentStore) DumpSSHCDN(id string) types.SSHCDNState {
	a.sshCDNAgentMapMu.Lock()
	agent := a.sshCDNAgentMap[id]
	defer a.sshCDNAgentMapMu.Unlock()
	if agent == nil {
		return types.SSHCDNState{
			AgentID: id,
		}
	}
	state := types.SSHCDNState{
		AgentID: id,
		SSHConn: types.Conn{
			LocaleAddr: agent.SSHConn.LocalAddr().String(),
			RemoteAddr: agent.SSHConn.RemoteAddr().String(),
		},
	}

	return state
}
