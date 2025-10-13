package store

import (
	"Goauld/common/types"
	"errors"
	"net"

	"github.com/quic-go/quic-go"
)

// QUICAgent holds the information of quic-connected agent.
type QUICAgent struct {
	QUICStream *quic.Stream
	SSHConn    net.Conn
}

// QuicAddAgent adds the TLS connections to the QUIC agent store.
func (a *AgentStore) QuicAddAgent(id string, quicAgent *QUICAgent) {
	a.quicAgentMapMu.Lock()
	a.quicAgentMap[id] = quicAgent
	a.quicAgentMapMu.Unlock()
}

// QuicRemoveAgent removes the TLS connections of the QUIC agent.
func (a *AgentStore) QuicRemoveAgent(id string) {
	a.quicAgentMapMu.Lock()
	delete(a.quicAgentMap, id)
	a.quicAgentMapMu.Unlock()
}

// QuicCloseAgent closes the TLS connections of the QUIC agent.
func (a *AgentStore) QuicCloseAgent(id string) error {
	a.quicAgentMapMu.Lock()
	agent := a.quicAgentMap[id]
	a.quicAgentMapMu.Unlock()
	a.QuicRemoveAgent(id)
	var errs []error
	if agent != nil && agent.QUICStream != nil {
		err := agent.SSHConn.Close()
		errs = append(errs, err)
	}
	if agent != nil && agent.QUICStream != nil {
		err := agent.SSHConn.Close()
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// DumpQUIC return the QUIC information associated to the agent.
func (a *AgentStore) DumpQUIC(id string) types.QUICState {
	a.quicAgentMapMu.Lock()
	agent := a.quicAgentMap[id]
	defer a.quicAgentMapMu.Unlock()
	state := types.QUICState{
		AgentID: id,
	}
	if agent == nil {
		return state
	}
	if agent.QUICStream != nil {
		state.QuicConn = types.Conn{}
	}
	if agent.SSHConn != nil {
		state.SSHConn = types.Conn{
			LocaleAddr: agent.SSHConn.LocalAddr().String(),
			RemoteAddr: agent.SSHConn.RemoteAddr().String(),
		}
	}

	return state
}
