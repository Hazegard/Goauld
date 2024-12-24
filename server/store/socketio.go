package store

import (
	"Goauld/server/db"
	sio "github.com/karagenc/socket.io-go"
)

func (a *AgentStore) SioAddAgent(agent *db.Agent, id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	a.sioAgentMap[id] = agent
	a.sioAgentMapMu.Unlock()
}
func (a *AgentStore) SioRemoveAgent(id sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	delete(a.sioAgentMap, id)
	a.sioAgentMapMu.Unlock()
}
func (a *AgentStore) SioGetAgent(id sio.ServerSocket) *db.Agent {
	a.sioAgentMapMu.Lock()
	agent := a.sioAgentMap[id]
	a.sioAgentMapMu.Unlock()
	return agent
}
