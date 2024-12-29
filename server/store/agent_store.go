package store

import (
	"Goauld/server/persistence"
	"errors"
	"fmt"
	sio "github.com/karagenc/socket.io-go"
	"sync"
)

var store *AgentStore
var once sync.Once

func NewAgentStore(_db *persistence.DB) *AgentStore {
	once.Do(func() {
		store = &AgentStore{
			db: _db,

			sioAgentMap:   make(map[sio.ServerSocket]*persistence.Agent),
			sioAgentMapMu: sync.Mutex{},

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
	db            *persistence.DB
	sioAgentMap   map[sio.ServerSocket]*persistence.Agent
	sioAgentMapMu sync.Mutex

	wsshAgentMap   map[string]*WsshAgent
	wsshAgentMapMu sync.Mutex

	sshttpAgentMap   map[string]*SSHTTPAgent
	sshttpAgentMapMu sync.Mutex

	tlsshAgentMap   map[string]*TLSSHAgent
	tlsshAgentMapMu sync.Mutex
}

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
