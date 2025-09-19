package store

import (
	"Goauld/common/types"
	"encoding/hex"
	"errors"
	"net"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

type SSHSession struct {
	Ctx       ssh.Context
	Conns     []*gossh.ServerConn
	Listeners []net.Listener
}

// AdSSHSession adds the SSH connections to the SSH agent store
func (a *AgentStore) AdSSHSession(id string, ctx ssh.Context, ln net.Listener) {
	a.sshAgentMapMu.Lock()
	sess := a.sshAgentMap[id]
	if sess == nil {
		sess = &SSHSession{
			Conns: nil,
		}
	}
	conn, ok := ctx.Value(ssh.ContextKeyConn).(*gossh.ServerConn)
	if ok {
		sess.Conns = append(sess.Conns, conn)
	}
	sess.Listeners = append(sess.Listeners, ln)
	a.sshAgentMap[id] = sess
	a.sshAgentMapMu.Unlock()
}

// SSHRemoveAgent removes the SSH connections of the SSH agent
func (a *AgentStore) SSHRemoveAgent(id string) {
	a.sshAgentMapMu.Lock()
	delete(a.sshAgentMap, id)
	a.sshAgentMapMu.Unlock()
}

// SSHCloseAgent closes the SSH connections of the SSH agent
func (a *AgentStore) SSHCloseAgent(id string) error {
	a.sshAgentMapMu.Lock()
	agent := a.sshAgentMap[id]
	a.sshAgentMapMu.Unlock()
	a.SSHRemoveAgent(id)
	var errs []error
	if agent != nil {
		for _, conn := range agent.Conns {
			err := conn.Close()
			errs = append(errs, err)
		}

		for _, ln := range agent.Listeners {
			err := ln.Close()
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

// DumpSSH return the SSH information associated with the agent
func (a *AgentStore) DumpSSH(id string) types.SSHState {
	a.sshAgentMapMu.Lock()
	agent := a.sshAgentMap[id]
	defer a.sshAgentMapMu.Unlock()
	state := types.SSHState{
		AgentId: id,
	}
	if agent == nil {
		return state
	}
	for _, conn := range agent.Conns {
		c := types.SSHConnection{

			AgentId: conn.User(),
			SshConn: types.Conn{
				RemoteAddr: conn.RemoteAddr().String(),
				LocaleAddr: conn.LocalAddr().String(),
			},
			ClientVersion: string(conn.ClientVersion()),
			SessionID:     hex.EncodeToString(conn.SessionID()),
			ServerVersion: string(conn.ServerVersion()),
		}
		state.SSHConnection = append(state.SSHConnection, c)
	}

	for _, ln := range agent.Listeners {
		state.SSHListeners = append(state.SSHListeners, ln.Addr().String())
	}

	return state
}
