package store

import (
	"net"
)

type SSHTTPAgent struct {
	SshConn net.Conn
}

func (a *AgentStore) SshttpAddAgent(ssh net.Conn, id string) {
	a.sshttpAgentMapMu.Lock()
	a.sshttpAgentMap[id] = &SSHTTPAgent{
		SshConn: ssh,
	}
	a.sshttpAgentMapMu.Unlock()
}

func (a *AgentStore) SshttpRemoveAgent(id string) {
	a.sshttpAgentMapMu.Lock()
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
}

func (a *AgentStore) SshttpGetAgent(id string) *SSHTTPAgent {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	a.sshttpAgentMapMu.Unlock()
	return agent
}

func (a *AgentStore) SshttpCloseAgent(id string) error {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[id]
	delete(a.sshttpAgentMap, id)
	a.sshttpAgentMapMu.Unlock()
	err := agent.SshConn.Close()
	a.SshttpRemoveAgent(id)
	return err
}
