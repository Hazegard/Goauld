package store

import (
	commonnet "Goauld/common/net"
	"errors"
	"fmt"
	"sync"

	socketio "Goauld/common/socket.io"
	"Goauld/common/types"
	"Goauld/common/utils"
	"Goauld/server/persistence"

	sio "github.com/karagenc/socket.io-go"
)

var (
	store *AgentStore
	once  sync.Once
)

// NewAgentStore saves in memory all the information related
// to the active connections
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

			quicAgentMap:   make(map[string]*QUICAgent),
			quicAgentMapMu: sync.Mutex{},

			tlsshAgentMap:   make(map[string]*TLSSHAgent),
			tlsshAgentMapMu: sync.Mutex{},

			sshAgentMap:   make(map[string]*SSHSession),
			sshAgentMapMu: sync.Mutex{},

			dnsshAgentMap:   make(map[string]*DNSSHAgent),
			dnsshAgentMapMu: sync.Mutex{},

			remoteAddrMap:   make(map[string]string),
			remoteAddrMapMu: sync.Mutex{},
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

	quicAgentMap   map[string]*QUICAgent
	quicAgentMapMu sync.Mutex

	tlsshAgentMap   map[string]*TLSSHAgent
	tlsshAgentMapMu sync.Mutex

	sshAgentMap   map[string]*SSHSession
	sshAgentMapMu sync.Mutex

	dnsshAgentMap   map[string]*DNSSHAgent
	dnsshAgentMapMu sync.Mutex

	remoteAddrMap   map[string]string
	remoteAddrMapMu sync.Mutex
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
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// ClearById Clears all agent connections related to a given agent id
func (a *AgentStore) ClearById(id string) error {
	return a.CloseAgentConnections(id)
}

// CloseAgentConnections closes all the connections of the agent
func (a *AgentStore) CloseAgentConnections(id string) error {
	return errors.Join(
		a.SSHCloseAgent(id),
		a.TlsshCloseAgent(id),
		a.WsshCloseAgent(id),
		a.SshttpCloseAgent(id),
		a.DnsshCloseAgent(id),
	)
}

// KillAGent kills the agent, if doKill is true, the agent does not restart
// If false, the agent resets and restarts
func (a *AgentStore) KillAGent(id string, doKill bool) error {
	socket := a.SioGetSocket(id)
	if socket == nil {
		err := a.db.SetAgentSshMode(id, "OFF", "")
		if err != nil {
			return fmt.Errorf("socket not found, error while disconnecting agent: %v", err)
		}
		return errors.New("socket not found")
	}

	socket.Emit(socketio.ExitEvent, doKill)
	return nil
}

// GetAllActivesId return the ID of all running agents
func (a *AgentStore) GetAllActivesId() []string {
	var ids []string
	a.tlsshAgentMapMu.Lock()
	for id := range a.tlsshAgentMap {
		ids = append(ids, id)
	}
	a.tlsshAgentMapMu.Unlock()

	a.quicAgentMapMu.Lock()
	for id := range a.quicAgentMap {
		ids = append(ids, id)
	}
	a.quicAgentMapMu.Unlock()

	a.wsshAgentMapMu.Lock()
	for id := range a.wsshAgentMap {
		ids = append(ids, id)
	}
	a.wsshAgentMapMu.Unlock()

	a.sshttpAgentMapMu.Lock()
	for id := range a.sshttpAgentMap {
		ids = append(ids, id)
	}
	a.sshttpAgentMapMu.Unlock()

	a.sioSocketMapMu.Lock()
	for id := range a.sioSocketMap {
		ids = append(ids, id)
	}
	a.sioSocketMapMu.Unlock()

	return utils.Unique(ids)
}

// GetAllStates returns the state of all agents
func (a *AgentStore) GetAllStates() []types.State {
	var states []types.State
	for _, id := range a.GetAllActivesId() {
		states = append(states, a.GetState(id))
	}
	return states
}

// GetState return the state of an agent
func (a *AgentStore) GetState(id string) types.State {
	state := types.State{
		Id:       id,
		TLSSH:    a.DumpTLSSH(id),
		QUIC:     a.DumpQUIC(id),
		WSSH:     a.DumpWSSH(id),
		SSHTTP:   a.DumpSSHTTP(id),
		SocketIO: a.DumpSocketIO(id),
		SSH:      a.DumpSSH(id),
		DNS:      a.DumpDNSSH(id),
	}
	return state
}

func (a *AgentStore) AddRemote(id string, remoteAddr string) {
	remoteIp, _ := commonnet.ExtractIP(remoteAddr)
	isLoopback := commonnet.IsLoopback(remoteIp)
	a.remoteAddrMapMu.Lock()
	defer a.remoteAddrMapMu.Unlock()
	if isLoopback {
		// If the new address is loopback, only set if none stored yet
		if a.remoteAddrMap[id] == "" {
			a.remoteAddrMap[id] = remoteIp
			return
		}
		// else do nothing (don't override)
		return
	}
	if remoteIp != "" {
		// If the new address is NOT loopback, always override
		a.remoteAddrMap[id] = remoteIp
	}
}

func (a *AgentStore) GetRemote(id string) string {
	a.remoteAddrMapMu.Lock()
	defer a.remoteAddrMapMu.Unlock()
	return a.remoteAddrMap[id]
}
