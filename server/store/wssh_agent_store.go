package store

import (
	"Goauld/server/db"
	"fmt"
	"net"
	"strings"
)

type WsshAgent struct {
	srcConn net.Conn
	dstConn net.Conn
	agent   *db.Agent
}

func (a *AgentStore) WsshAddAgent(agent *db.Agent, id string, src net.Conn, dst net.Conn) {
	a.wsshAgentMapMu.Lock()
	a.wsshAgentMap[id] = &WsshAgent{srcConn: src, dstConn: dst, agent: agent}
	a.wsshAgentMapMu.Unlock()
}

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

	return fmt.Errorf(strings.Join(errs, " / "))
}

func (a *AgentStore) WsshRemoveAgent(id string) {
	a.wsshAgentMapMu.Lock()
	delete(a.wsshAgentMap, id)
	a.wsshAgentMapMu.Unlock()
}

func (a *AgentStore) WsshGetAgent(id string) *WsshAgent {
	a.wsshAgentMapMu.Lock()
	agent := a.wsshAgentMap[id]
	a.wsshAgentMapMu.Unlock()
	return agent
}
