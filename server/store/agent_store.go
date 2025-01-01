package store

import (
	socketio "Goauld/common/socket.io"
	"Goauld/server/persistence"
	"errors"
	"fmt"
	sio "github.com/karagenc/socket.io-go"
	"sync"
)

var store *AgentStore
var once sync.Once

// NewAgentStore saves in memory all the information related
// to the actives connections
func NewAgentStore(_db *persistence.DB) *AgentStore {
	once.Do(func() {
		store = &AgentStore{
			db: _db,

			sioAgentMap:    make(map[sio.ServerSocket]*persistence.Agent),
			sioAgentMapMu:  sync.Mutex{},
			sioSocketMap:   make(map[string]sio.ServerSocket),
			sioSocketMapMu: sync.Mutex{},

			wsshAgentMap:   make(map[string]*WsshAgent),
			wsshAgentMapMu: sync.Mutex{},

			sshttpAgentMap:   make(map[string]*SSHTTPAgent),
			sshttpAgentMapMu: sync.Mutex{},

			tlsshAgentMap:   make(map[string]*TLSSHAgent),
			tlsshAgentMapMu: sync.Mutex{},
		}
	})
	return store
}

type AgentStore struct {
	db             *persistence.DB
	sioAgentMap    map[sio.ServerSocket]*persistence.Agent
	sioAgentMapMu  sync.Mutex
	sioSocketMap   map[string]sio.ServerSocket
	sioSocketMapMu sync.Mutex

	wsshAgentMap   map[string]*WsshAgent
	wsshAgentMapMu sync.Mutex

	sshttpAgentMap   map[string]*SSHTTPAgent
	sshttpAgentMapMu sync.Mutex

	tlsshAgentMap   map[string]*TLSSHAgent
	tlsshAgentMapMu sync.Mutex
}

// ClearByPort Clears all agent connections related to a given port
func (a *AgentStore) ClearByPort(port int) error {
	agents, err := a.db.GetAgentsByUsedPort(port)
	if err != nil {
		return fmt.Errorf("get agents by used port:%d err:%v", port, err)
	}

	errs := make([]error, 0)

	for _, agent := range agents {
		err := a.ClearById(agent.Id)
		if err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// ClearById Clears all agent connections related to a given agent id
func (a *AgentStore) ClearById(id string) error {
	errs := make([]error, 0)
	err := a.TlsshCloseAgent(id)
	if err != nil {
		errs = append(errs, err)
	}
	err = a.SshttpCloseAgent(id)
	if err != nil {
		errs = append(errs, err)
	}
	err = a.SshttpCloseAgent(id)
	if err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// CloseAgentConnections closes all the connections of the agent
func (a *AgentStore) CloseAgentConnections(id string) error {
	err1 := a.WsshCloseAgent(id)
	err2 := a.SshttpCloseAgent(id)
	err3 := a.TlsshCloseAgent(id)
	return errors.Join(err1, err2, err3)
}

func (a *AgentStore) KillAGent(id string) error {
	socket := a.SioGetSocket(id)
	if socket == nil {
		return errors.New("socket not found")
	}
	socket.Emit(socketio.ExitEvent)
	return nil
}
