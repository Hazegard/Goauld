package store

import (
	"net"

	"Goauld/common/types"
)

type SSHTTPAgent struct {
	SshConn net.Conn
}

// SshttpAddAgent adds the ssh connection of agent id to the agent store
func (a *AgentStore) SshttpAddAgent(ssh net.Conn, id string) {
	a.sshttpAgentMapMu.Lock()
	a.sshttpAgentMap[id] = &SSHTTPAgent{
		SshConn: ssh,
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
		err = agent.SshConn.Close()
	}
	a.SshttpRemoveAgent(id)
	return err
}

// DumpSSHTTP return the SSHTTP information associated to the agent
func (a *AgentStore) DumpSSHTTP(id string) types.SSHTTState {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	defer a.sshttpAgentMapMu.Unlock()
	state := types.SSHTTState{
		AgentId:       id,
		SSHRemoteAddr: agent.SshConn.RemoteAddr().String(),
		SSHLocaleAddr: agent.SshConn.LocalAddr().String(),
	}
	return state
}
