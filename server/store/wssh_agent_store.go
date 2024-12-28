package store

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type WsshAgent struct {
	srcConn net.Conn
	dstConn net.Conn
}

func (a *AgentStore) WsshAddAgent(id string, src net.Conn, dst net.Conn) {
	a.wsshAgentMapMu.Lock()
	a.wsshAgentMap[id] = &WsshAgent{srcConn: src, dstConn: dst}
	a.wsshAgentMapMu.Unlock()
}

func (a *AgentStore) CloseAgentConnections(id string) error {
	err1 := a.WsshCloseAgent(id)
	err2 := a.SshttpCloseAgent(id)
	err3 := a.TlsshCloseAgent(id)
	return errors.Join(err1, err2, err3)
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
