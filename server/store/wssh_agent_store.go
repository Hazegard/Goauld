package store

import (
	"errors"
	"net"
	"strings"
)

type WsshAgent struct {
	srcConn net.Conn
	dstConn net.Conn
}

// WsshAddAgent adds the WSSH connections to the store
func (a *AgentStore) WsshAddAgent(id string, src net.Conn, dst net.Conn) {
	a.wsshAgentMapMu.Lock()
	a.wsshAgentMap[id] = &WsshAgent{srcConn: src, dstConn: dst}
	a.wsshAgentMapMu.Unlock()
}

// WsshCloseAgent closes the WSSH connections
func (a *AgentStore) WsshCloseAgent(id string) error {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	a.wsshAgentMapMu.Unlock()
	a.WsshRemoveAgent(id)
	errs := []string{}
	if agent != nil && agent.srcConn != nil {
		err1 := agent.srcConn.Close()
		if err1 != nil {
			errs = append(errs, err1.Error())
		}
	}
	if agent != nil && agent.dstConn != nil {
		err2 := agent.dstConn.Close()
		if err2 != nil {
			errs = append(errs, err2.Error())
		}
	}

	return errors.New(strings.Join(errs, " / "))
}

// WsshRemoveAgent remove the agent of the store
func (a *AgentStore) WsshRemoveAgent(id string) {
	a.wsshAgentMapMu.Lock()
	delete(a.wsshAgentMap, id)
	a.wsshAgentMapMu.Unlock()
}

// WsshGetAgent returns the WSSH connections of the agent
func (a *AgentStore) WsshGetAgent(id string) *WsshAgent {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	a.wsshAgentMapMu.Unlock()
	return agent
}
