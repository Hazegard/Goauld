package store

import (
	"Goauld/common/types"
	"github.com/xtaci/smux"
	"net"
)

type DNSSHAgent struct {
	SSHConn net.Conn
	Session *smux.Stream
	KcpAddr string
}

// DnsshAddAgent adds the ssh connection of agent id to the agent store
func (a *AgentStore) DnsshAddAgent(ssh net.Conn, session *smux.Stream, kcpAddr string, id string) {
	a.dnsshAgentMapMu.Lock()
	a.dnsshAgentMap[id] = &DNSSHAgent{
		SSHConn: ssh,
		Session: session,
		KcpAddr: kcpAddr,
	}
	a.dnsshAgentMapMu.Unlock()
}

// DnsshRemoveAgent removes the ssh connection of the agent id to the agent store
func (a *AgentStore) DnsshRemoveAgent(id string) {
	a.dnsshAgentMapMu.Lock()
	delete(a.dnsshAgentMap, id)
	a.dnsshAgentMapMu.Unlock()
}

// DnsshGetAgent returns the ssh connection of the agent id to the agent store
func (a *AgentStore) DnsshGetAgent(id string) *DNSSHAgent {
	a.dnsshAgentMapMu.Lock()
	agent := a.dnsshAgentMap[id]
	a.dnsshAgentMapMu.Unlock()
	return agent
}

// DnsshCloseAgent closes the ssh connection of the agent id to the agent store
func (a *AgentStore) DnsshCloseAgent(id string) error {
	var err error
	a.dnsshAgentMapMu.Lock()
	agent := a.dnsshAgentMap[id]
	delete(a.dnsshAgentMap, id)
	a.dnsshAgentMapMu.Unlock()
	if agent != nil && agent.SSHConn != nil {
		err = agent.SSHConn.Close()
	}
	a.DnsshRemoveAgent(id)
	return err
}

// DumpDNSSH return the DNSSH information associated to the agent
func (a *AgentStore) DumpDNSSH(id string) types.DNSSHState {
	a.dnsshAgentMapMu.Lock()
	agent := a.dnsshAgentMap[id]
	defer a.dnsshAgentMapMu.Unlock()
	if agent == nil {
		return types.DNSSHState{
			AgentId: id,
		}
	}

	state := types.DNSSHState{
		AgentId:              id,
		SSHRemoteAddr:        agent.SSHConn.RemoteAddr().String(),
		SSHLocaleAddr:        agent.SSHConn.LocalAddr().String(),
		KCPAddr:              agent.KcpAddr,
		MuxSessionLocaleAddr: agent.Session.LocalAddr().String(),
		MuxSessionRemoteAddr: agent.Session.RemoteAddr().String(),
	}
	return state
}
