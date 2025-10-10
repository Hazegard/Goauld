package store

import (
	"Goauld/common/types"
	"Goauld/server/persistence"

	sio "github.com/hazegard/socket.io-go"
)

// SioAddAgent adds the agent to the associated socket
func (a *AgentStore) SioAddAgent(agent *persistence.Agent, socket sio.ServerSocket) {
	a.sioAgentMapMu.Lock()
	a.sioAgentMap[socket] = agent
	a.sioAgentMapMu.Unlock()

	a.sioSocketMapMu.Lock()
	a.sioSocketMap[agent.Id] = socket
	a.sioSocketMap[agent.Name] = socket
	a.sioSocketMapMu.Unlock()
}

// SioRemoveAgent removes the agent of the associated socket
func (a *AgentStore) SioRemoveAgent(socket sio.ServerSocket) {
	agent := a.SioGetAgent(socket)

	a.sioAgentMapMu.Lock()
	delete(a.sioAgentMap, socket)
	a.sioAgentMapMu.Unlock()

	a.sioSocketMapMu.Lock()
	delete(a.sioSocketMap, agent.Id)
	delete(a.sioSocketMap, agent.Name)
	a.sioSocketMapMu.Unlock()
}

// SioGetAgent returns the agent of the associated socket
func (a *AgentStore) SioGetAgent(socket sio.ServerSocket) *persistence.Agent {
	a.sioAgentMapMu.Lock()
	agent := a.sioAgentMap[socket]
	a.sioAgentMapMu.Unlock()
	if agent == nil {
		return &persistence.Agent{}
	}
	return agent
}

// SioGetSocket returns the socket of the associated store
func (a *AgentStore) SioGetSocket(id string) sio.ServerSocket {
	a.sioSocketMapMu.Lock()
	socket := a.sioSocketMap[id]
	a.sioSocketMapMu.Unlock()
	if socket == nil {
		return nil
	}
	return socket
}

// DumpSocketIO return the socket.io information associated with the agent
func (a *AgentStore) DumpSocketIO(id string) types.SocketIOState {
	a.sioSocketMapMu.Lock()
	socket := a.sioSocketMap[id]
	defer a.sioSocketMapMu.Unlock()
	state := types.SocketIOState{
		AgentId: id,
	}
	if socket != nil {
		state.SocketId = string(socket.ID())
		state.Connected = socket.Connected()
		state.Recovered = socket.Recovered()
	}

	return state
}
