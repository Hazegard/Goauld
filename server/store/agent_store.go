package store

import (
	"Goauld/server/db"
	sio "github.com/karagenc/socket.io-go"
	"sync"
)

var store *AgentStore
var once sync.Once

func NewAgentStore() *AgentStore {
	once.Do(func() {
		store = &AgentStore{
			sioAgentMap:   make(map[sio.ServerSocket]*db.Agent),
			sioAgentMapMu: sync.Mutex{},

			wsshAgentMap:   make(map[string]*WsshAgent),
			wsshAgentMapMu: sync.Mutex{},

			sshttpAgentMap:   make(map[sio.ServerSocket]*SSHTTPAgent),
			sshttpAgentMapMu: sync.Mutex{},
		}
	})
	return store
}

type AgentStore struct {
	sioAgentMap   map[sio.ServerSocket]*db.Agent
	sioAgentMapMu sync.Mutex

	wsshAgentMap   map[string]*WsshAgent
	wsshAgentMapMu sync.Mutex

	sshttpAgentMap   map[sio.ServerSocket]*SSHTTPAgent
	sshttpAgentMapMu sync.Mutex
}
