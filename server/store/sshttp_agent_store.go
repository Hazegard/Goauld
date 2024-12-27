package store

import (
	sio "github.com/karagenc/socket.io-go"
	"net"
)

type SSHTTPAgent struct {
	sshConn net.Conn
	socket  sio.ServerSocket
}

func (a *AgentStore) SshttpAddAgent(ssh net.Conn, socket sio.ServerSocket) {
	a.sshttpAgentMapMu.Lock()
	a.sshttpAgentMap[socket] = &SSHTTPAgent{
		sshConn: ssh,
		socket:  socket,
	}
	a.sshttpAgentMapMu.Unlock()
}

func (a *AgentStore) SshttpRemoveAgent(socket sio.ServerSocket) {
	a.sshttpAgentMapMu.Lock()
	delete(a.sshttpAgentMap, socket)
	a.sshttpAgentMapMu.Unlock()
}

func (a *AgentStore) SshttpGetAgent(socket sio.ServerSocket) *SSHTTPAgent {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[socket]
	a.sshttpAgentMapMu.Unlock()
	return agent
}

func (a *AgentStore) SshttpCloseAgent(socket sio.ServerSocket) error {
	a.sshttpAgentMapMu.Lock()
	agent := a.sshttpAgentMap[socket]
	delete(a.sshttpAgentMap, socket)
	a.SshttpRemoveAgent(socket)
	err := agent.sshConn.Close()
	a.sshttpAgentMapMu.Unlock()
	//agent.socket.Disconnect(false)
	return err
}
