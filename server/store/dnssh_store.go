package store

import (
	"Goauld/common/types"
	"errors"
	"github.com/xtaci/smux"
	"net"
)

type DNSSHAgent struct {
	UpstreamConn []net.Conn
	Session      *smux.Stream
	KcpAddr      string
}

// TODO: ajouter les conn pour le serveur SSH et pour le serveur Websocket
// idée: faire une map des upstreamm conn address pour garder l'implémentation simple et flexible ?

// DnsshAddAgent adds the ssh connection of agent id to the agent store
func (a *AgentStore) DnsshAddAgent(upstreamConn net.Conn, session *smux.Stream, kcpAddr string, id string) {
	a.dnsshAgentMapMu.Lock()
	agent, ok := a.dnsshAgentMap[id]
	if !ok {
		agent = &DNSSHAgent{
			UpstreamConn: []net.Conn{upstreamConn},
			Session:      session,
			KcpAddr:      kcpAddr,
		}
	} else {
		agent.UpstreamConn = append(agent.UpstreamConn, upstreamConn)
	}
	a.dnsshAgentMap[id] = agent
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
	var errs []error
	a.dnsshAgentMapMu.Lock()
	agent := a.dnsshAgentMap[id]
	delete(a.dnsshAgentMap, id)
	a.dnsshAgentMapMu.Unlock()
	if agent != nil {
		for _, conn := range agent.UpstreamConn {
			if conn != nil {
				errs = append(errs, conn.Close())

			}
		}
	}
	a.DnsshRemoveAgent(id)
	return errors.Join(errs...)
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

	upstreamConn := make([]types.Conn, len(agent.UpstreamConn))
	for i, conn := range agent.UpstreamConn {
		upstreamConn[i] = types.Conn{
			RemoteAddr: conn.RemoteAddr().String(),
			LocaleAddr: conn.LocalAddr().String(),
		}
	}
	state := types.DNSSHState{
		AgentId:      id,
		UpstreamConn: upstreamConn,
		KCPAddr:      agent.KcpAddr,
		MuxSession: types.Conn{
			RemoteAddr: agent.Session.RemoteAddr().String(),
			LocaleAddr: agent.Session.LocalAddr().String(),
		},
	}
	return state
}
