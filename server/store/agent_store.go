// Package store holds the dynamic agents information
package store

import (
	commonnet "Goauld/common/net"
	"errors"
	"fmt"
	"sync"

	"Goauld/common/types"
	"Goauld/common/utils"
	"Goauld/server/persistence"

	socketio "Goauld/common/socket.io"

	sio "github.com/hazegard/socket.io-go"
)

var (
	store *AgentStore
	once  sync.Once
)

// NewAgentStore saves in memory all the information related
// to the active connections.
func NewAgentStore(_db *persistence.DB) *AgentStore {
	once.Do(func() {
		store = &AgentStore{
			db: _db,

			sioAgentMap:    make(map[sio.ServerSocket]*persistence.Agent),
			sioAgentMapMu:  sync.Mutex{},
			sioSocketMap:   make(map[string]sio.ServerSocket),
			sioSocketMapMu: sync.Mutex{},

			wsshAgentMap:   make(map[string]*WSSHAgent),
			wsshAgentMapMu: sync.Mutex{},

			sshttpAgentMap:   make(map[string]*SSHTTPAgent),
			sshttpAgentMapMu: sync.Mutex{},

			sshCDNAgentMap:   make(map[string]*SSHCDNAgent),
			sshCDNAgentMapMu: sync.Mutex{},

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

// AgentStore holds the information of all connected agents, depending on the tunnel type.
type AgentStore struct {
	db             *persistence.DB
	sioAgentMap    map[sio.ServerSocket]*persistence.Agent
	sioAgentMapMu  sync.Mutex
	sioSocketMap   map[string]sio.ServerSocket
	sioSocketMapMu sync.Mutex

	wsshAgentMap   map[string]*WSSHAgent
	wsshAgentMapMu sync.Mutex

	sshttpAgentMap   map[string]*SSHTTPAgent
	sshttpAgentMapMu sync.Mutex

	sshCDNAgentMap   map[string]*SSHCDNAgent
	sshCDNAgentMapMu sync.Mutex

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

// ClearByPort Clears all agent connections related to a given port.
func (a *AgentStore) ClearByPort(port int) error {
	agents, err := a.db.GetAgentsByUsedPort(port)
	if err != nil {
		return fmt.Errorf("get agents by used port:%d err:%w", port, err)
	}

	errs := make([]error, 0)

	for _, agent := range agents {
		err := a.ClearByID(agent.ID)
		errs = append(errs, err)
	}

	return errors.Join(errs...)
}

// ClearByID Clears all agent connections related to a given agent id.
func (a *AgentStore) ClearByID(id string) error {
	return a.CloseAgentConnections(id)
}

// CloseAgentConnections closes all the connections of the agent.
func (a *AgentStore) CloseAgentConnections(id string) error {
	return errors.Join(
		a.SSHCloseAgent(id),
		a.TLSSHCloseAgent(id),
		a.WSSHCloseAgent(id),
		a.SSHTTPCloseAgent(id),
		a.DnsshCloseAgent(id),
	)
}

// IsAgentConnected returns whether an agent is connected to the server.
func (a *AgentStore) IsAgentConnected(id string) bool {
	a.sioSocketMapMu.Lock()
	socket, ok := a.sioSocketMap[id]
	a.sioSocketMapMu.Unlock()
	if socket == nil || !ok {
		return false
	}
	a.sioAgentMapMu.Lock()
	agent, ok := a.sioAgentMap[socket]
	a.sioAgentMapMu.Unlock()
	if agent == nil || !ok {
		return false
	}

	a.dnsshAgentMapMu.Lock()
	dnsAgent := a.dnsshAgentMap[id]
	a.dnsshAgentMapMu.Unlock()
	if dnsAgent != nil {
		return true
	}

	a.sshAgentMapMu.Lock()
	sshAgent := a.sshAgentMap[id]
	a.sshAgentMapMu.Unlock()
	if sshAgent != nil {
		return true
	}

	a.sshttpAgentMapMu.Lock()
	sshttpAgent := a.sshttpAgentMap[id]
	a.sshttpAgentMapMu.Unlock()
	if sshttpAgent != nil {
		return true
	}

	a.wsshAgentMapMu.Lock()
	wsshAgent := a.sshttpAgentMap[id]
	a.wsshAgentMapMu.Unlock()
	if wsshAgent != nil {
		return true
	}

	a.tlsshAgentMapMu.Lock()
	tlsshAgent := a.sshttpAgentMap[id]
	a.tlsshAgentMapMu.Unlock()
	if tlsshAgent != nil {
		return true
	}

	a.quicAgentMapMu.Lock()
	quickAgent := a.sshttpAgentMap[id]
	a.quicAgentMapMu.Unlock()

	return quickAgent != nil
}

// KillAGent kills the agent, if doKill is true, the agent does not restart
// If false, the agent resets and restarts.
func (a *AgentStore) KillAGent(id string, doKill bool) error {
	socket := a.SioGetSocket(id)
	if socket == nil {
		err := a.db.SetAgentSSHMode(id, "OFF", "")
		if err != nil {
			return fmt.Errorf("socket not found, error while disconnecting agent: %w", err)
		}

		return errors.New("socket not found")
	}

	socket.Emit(socketio.ExitEvent.ID(), doKill)

	return nil
}

// GetAllActivesID return the ID of all running agents.
func (a *AgentStore) GetAllActivesID() []string {
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

// GetAllStates returns the state of all agents.
func (a *AgentStore) GetAllStates() []types.State {
	var states []types.State
	for _, id := range a.GetAllActivesID() {
		states = append(states, a.GetState(id))
	}

	return states
}

// GetState return the state of an agent.
func (a *AgentStore) GetState(id string) types.State {
	state := types.State{
		ID:       id,
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

// AddRemote add the remote address of a newly connected agent.
func (a *AgentStore) AddRemote(id string, remoteAddr string) {
	remoteIP, _ := commonnet.ExtractIP(remoteAddr)
	isLoopback := commonnet.IsLoopback(remoteIP)
	a.remoteAddrMapMu.Lock()
	defer a.remoteAddrMapMu.Unlock()
	if isLoopback {
		// If the new address is loopback, only set if none stored yet
		if a.remoteAddrMap[id] == "" {
			a.remoteAddrMap[id] = remoteIP

			return
		}
		// else do nothing (don't override)
		return
	}
	if remoteIP != "" {
		// If the new address is NOT loopback, always override
		a.remoteAddrMap[id] = remoteIP
	}
}

// GetRemote returns the public remote address from which the agent connects.
func (a *AgentStore) GetRemote(id string) string {
	a.remoteAddrMapMu.Lock()
	defer a.remoteAddrMapMu.Unlock()

	return a.remoteAddrMap[id]
}
