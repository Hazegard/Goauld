package store

import (
	"Goauld/server/persistence"
	sio "github.com/karagenc/socket.io-go"
)

func (a *AgentStore) SioAddAgent(agent *persistence.Agent, id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	a.sioAgentMap[id] = agent
	a.sioAgentMapMu.Unlock()
}
func (a *AgentStore) SioRemoveAgent(id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	delete(a.sioAgentMap, id)
	a.sioAgentMapMu.Unlock()
}
func (a *AgentStore) SioGetAgent(id sio.ServerSocket) *persistence.Agent {
	a.sioAgentMapMu.Lock()
	agent := a.sioAgentMap[id]
	a.sioAgentMapMu.Unlock()
	if agent == nil {
		return &persistence.Agent{}
	}
	return agent
}
