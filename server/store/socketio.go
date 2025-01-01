package store

import (
	"Goauld/server/persistence"
	sio "github.com/karagenc/socket.io-go"
)

// SioAddAgent adds the agent to the associated store
func (a *AgentStore) SioAddAgent(agent *persistence.Agent, id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	a.sioAgentMap[id] = agent
	a.sioAgentMapMu.Unlock()
}

// SioRemoveAgent removes the agent of the associated store
func (a *AgentStore) SioRemoveAgent(id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	delete(a.sioAgentMap, id)
	a.sioAgentMapMu.Unlock()
}

// SioGetAgent returns the agent of the associated store
func (a *AgentStore) SioGetAgent(id sio.ServerSocket) *persistence.Agent {
	a.sioAgentMapMu.Lock()
	agent := a.sioAgentMap[id]
	a.sioAgentMapMu.Unlock()
	if agent == nil {
		return &persistence.Agent{}
	}
	return agent
}
