// Package sshd holds the SSHD server
package sshd

import (
	"Goauld/common/log"
	_net "Goauld/common/net"
	_ssh "Goauld/common/ssh"
	"Goauld/common/types"
	"Goauld/server/config"
	"Goauld/server/control"
	"Goauld/server/persistence"
	"Goauld/server/store"
	"context"
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/ssh"

	gossh "golang.org/x/crypto/ssh"
)

// StartSshd init and start the sshd server.
func StartSshd(context context.Context, db *persistence.DB, store *store.AgentStore) {
	listener, err := net.Listen("tcp", config.Get().LocalSSHAddr())
	if err != nil {
		panic(err)
	}
	// Update if listening on 0 to get the real port
	//nolint:forcetypeassert
	//nolint:forcetypeassert
	config.Get().UpdateSSHAddr(listener.Addr().(*net.TCPAddr).Port)
	log.Info().Str("Address", config.Get().LocalSSHAddr()).Msgf("SSH server listening")
	forwardHandler := &ForwardedTCPHandler{}

	// The SSHD server
	var s = &ssh.Server{
		/*KeyboardInteractiveHandler: func(ctx ssh.Context, challenger gossh.KeyboardInteractiveChallenge) bool {
			answers, err := challenger()
		},*/
		Version: "Server",
		Handler: func(s ssh.Session) {
			srcAddr := s.RemoteAddr().String()
			log.Error().Str("User", s.User()).Msgf("New ssh connection from %s", srcAddr)

			log.Error().Str("User", s.User()).Msgf("START SSH connection from: %s", s.RemoteAddr().String())
			defer log.Error().Str("User", s.User()).Msgf("END  SSH connection from: %s", s.LocalAddr().String())
		},
		LocalPortForwardingCallback: func(ctx ssh.Context, _ string, destinationPort uint32) bool {
			// TODO: should we check fo destination to be localhost ?
			username := ctx.User()
			sourceIP := strings.Split(ctx.RemoteAddr().String(), ":")[0]
			log.Trace().Str("User", username).Str("Port", strconv.Itoa(int(destinationPort))).Msgf("SSH local Port forwarding attempt from: %s", ctx.RemoteAddr().String())
			if !_net.IsIPAllowed(sourceIP, config.Get().AllowedIPs) {
				log.Warn().Err(errors.New("ip not in whitelist")).Str("Source IP", sourceIP).Str("Agent.Name", username).Msg("unable to port forward")

				return false
			}
			agent, err := db.FindAgentByName(username)
			if err != nil {
				log.Warn().Err(err).Str("User", username).Str("Port", strconv.Itoa(int(destinationPort))).Msg("port forward failed, unable to find agent")

				return false
			}
			if !agent.IsPortForwarded(int(destinationPort)) {
				log.Warn().Err(errors.New("attempt to forward forbidden port")).Str("Port", strconv.Itoa(int(destinationPort))).Msg("port forward failed")

				return false
			}

			return true
		},
		// Handle the reverse port forwarding, it gets the agent from the database
		// and update the agent to add the port in the database
		// If not agent is retrieved from the database, cancel the port forwarding
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Trace().Str("User", ctx.User()).Str("Port", strconv.Itoa(int(port))).Msgf("Reverse port forward to %s", host)
			id := ctx.User()
			err := db.AddPortToAgent(id, int(port))
			if err != nil {
				log.Error().Err(err).Str("User", ctx.User()).Msg("Failed to add port to agent")

				return false
			}

			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
				log.Trace().Str("User", ctx.User()).Str("Type", req.Type).Str("Payload", string(req.Payload)).Msg("SSH Request received")

				ok, payload, ln := forwardHandler.HandleSSHRequest(ctx, srv, req)
				if ln != nil {
					store.AdSSHSession(ctx.User(), ctx, ln)
				}

				return ok, payload
			},
			"cancel-tcpip-forward": func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
				ok, payload, _ := forwardHandler.HandleSSHRequest(ctx, srv, req)

				return ok, payload
			},
			// HandleKeepAlive returns pong when an agent sends a ping
			// This ping pong mechanism is used to perform a keepalive of the connections
			"ping":                  HandleKeepAlive(db),
			"keepalive@openssh.com": HandleKeepAlive(db),
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session": func(srv *ssh.Server, conn *gossh.ServerConn, newChan gossh.NewChannel, ctx ssh.Context) {
				log.Debug().Str("User", ctx.User()).Msg("New session")
				ssh.DefaultSessionHandler(srv, conn, newChan, ctx)
			},
		},
		// PublicKeyHandler handles the public key authentication
		// the username connecting is the id of the agent
		// The authentication only succeeds if
		// - The username matches the ID of an already registered agent
		// - The agent has a public key configured
		// - The public key matched
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			id := ctx.User()
			remote := ctx.RemoteAddr().String()
			log.Debug().Str("User", id).Str("Remote", remote).Msgf("SSH Connection attempt")
			agent, err := db.FindAgentByID(id)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", id)

				return false
			}
			log.Trace().Str("User", id).Msg("Agent found, getting public key...")

			agentPubKey, err := _ssh.ParseSSHPublicKey(agent.PublicKey)
			if err != nil {
				log.Debug().Msgf("Error parsing public key (%s)", id)

				return false
			}
			log.Trace().Str("User", id).Msg("Public Key found, checking public key...")
			if !ssh.KeysEqual(agentPubKey, key) {
				log.Warn().Str("User", id).Str("Remote", remote).Msg("Wrong Public Key...")

				return false
			}

			log.Trace().Msgf("SSH connection succeeded from %s (%s)", ctx.User(), id)
			err = db.SetAgentSSHMode(id, "SSH", remote)
			if err != nil {
				log.Warn().Str("User", id).Str("Remote", remote).Str("SSH Mode", "SSH").Msg("Error updating connection mode...")
			}

			return true
		},
		PasswordHandler: func(ctx ssh.Context, inPwd string) bool {
			sourceIP := strings.Split(ctx.RemoteAddr().String(), ":")[0]
			log.Trace().Str("User", ctx.User()).Str("IP", sourceIP).Msg("SSH Connection attempt")
			if !_net.IsIPAllowed(sourceIP, config.Get().AllowedIPs) {
				log.Trace().Str("Remote", sourceIP).Msg("Connection attempt from non whitelisted IP address")

				return false
			}

			pwd := types.ServerToAgentPassword{}
			err := json.Unmarshal([]byte(inPwd), &pwd)
			if err != nil {
				log.Warn().Err(err).Str("User", ctx.User()).Msg("Error parsing agent password")

				return false
			}

			agentName := ctx.User()
			agent, err := db.FindAgentByName(agentName)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", agentName)

				return false
			}
			if agent.SSHMode == "/" {
				log.Info().Str("Agent.Name", agent.Name).Msg("Agent not connected")

				return false
			}

			isStaticPwdValid := control.ValidateStaticPassword(agent, store.SioGetSocket(agent.ID), pwd.HashAgentPassword)
			if !isStaticPwdValid {
				return false
			}

			agent.Source = sourceIP
			err = db.ValidatePasswordAndRotateIfTrue(agent.ID, pwd.ServerPassword)
			// err = agent.ValidatePasswordAndRotateIfTrue(password)
			if err != nil {
				log.Warn().Err(err).Str("Incoming", pwd.ServerPassword).Str("Agent.Name", agentName).Str("Agent.ID", agent.ID).Msg("Failed to validate agent password")

				return false
			}
			err = db.UpdateAgentFieldShadow(agent, "OneTimePassword")
			if err != nil {
				log.Warn().Err(err).Str("Agent.Name", agentName).Str("Agent.ID", agent.ID).Msg("Failed to update agent password")

				return false
			}
			log.Trace().Str("Agent.Name", agentName).Msg("Password accepted")

			return true
		},
		// SessionRequestCallback logs information when a user requests a session
		SessionRequestCallback: func(sess ssh.Session, _ string) bool {
			id := sess.User()
			remote := sess.RemoteAddr().String()

			log.Info().Str("User", id).Str("Remote", remote).Msgf("SSH session requested from %s", id)

			return false
		},
	}
	err = s.Serve(listener)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	log.Info().Msg("SSH server stopped")
	context.Done()
}

//nolint:revive
func HandleKeepAlive(db *persistence.DB) func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	return func(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
		log.Trace().Str("User", ctx.User()).Msg("PING received")

		id := ctx.User()
		agent, err := db.FindAgentByID(id)
		if err != nil {
			log.Error().Err(err).Str("User", id).Msg("Failed to find agent")

			return true, []byte("pong")
		}
		agent.LastPing = time.Now()
		_ = db.UpdateAgentFieldShadow(agent, "LastPing")

		return true, []byte("pong")
	}
}
