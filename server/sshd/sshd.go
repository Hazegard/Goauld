package sshd

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"

	"Goauld/common/log"
	_net "Goauld/common/net"
	_ssh "Goauld/common/ssh"
	"Goauld/server/config"
	"Goauld/server/persistence"

	"github.com/gliderlabs/ssh"

	gossh "golang.org/x/crypto/ssh"
)

// StartSshd init and start the sshd server
func StartSshd(context context.Context, db *persistence.DB) {
	listener, err := net.Listen("tcp", config.Get().LocalSShAddr())
	if err != nil {
		panic(err)
	}
	// Update if listening on 0 to get the real port
	config.Get().UpdateSSHAddr(listener.Addr().(*net.TCPAddr).Port)
	log.Info().Str("Address", config.Get().LocalSShAddr()).Msgf("SSH server listening")
	forwardHandler := &ssh.ForwardedTCPHandler{}

	// The SSHD server
	s := &ssh.Server{
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
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			username := ctx.User()
			sourceIp := strings.Split(ctx.RemoteAddr().String(), ":")[0]
			log.Trace().Str("User", username).Str("Port", strconv.Itoa(int(destinationPort))).Msgf("SSH local Port forwarding attempt from: %s", ctx.RemoteAddr().String())
			if !_net.IsIPAllowed(sourceIp, config.Get().AllowedIPs) {
				log.Warn().Err(errors.New("ip not in whitelist")).Str("Source IP", sourceIp).Str("Agent.Name", username).Msg("unable to port forward")
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
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			"ping":                 handlePing,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
		},
		// PublicKeyHandler handles the public key authentication
		// the username connecting is the id of the agent
		// The authentication only succeed if
		// - The username matches the ID of an already registered agent
		// - The agent has a public key configured
		// - The public key matched
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			id := ctx.User()
			remote := ctx.RemoteAddr().String()
			log.Debug().Str("User", id).Str("Remote", remote).Msgf("SSH Connection attempt")
			agent, err := db.FindAgentById(id)
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
			err = db.SetAgentSshMode(id, "SSH")
			if err != nil {
				log.Warn().Str("User", id).Str("Remote", remote).Str("SSH Mode", "SSH").Msg("Error updating connection mode...")
			}
			return true
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			sourceIp := strings.Split(ctx.RemoteAddr().String(), ":")[0]
			if !_net.IsIPAllowed(sourceIp, config.Get().AllowedIPs) {
				log.Trace().Str("Remote", sourceIp).Msg("Connection attempt from non whitelisted IP address")
				return false
			}
			agentName := ctx.User()
			agent, err := db.FindAgentByName(agentName)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", agentName)
				return false
			}
			if agent.SshMode == "/" {
				log.Info().Str("Agent.Name", agent.Name).Msg("Agent not connected")
				return false
			}
			agent.Source = sourceIp
			err = agent.ValidatePasswordAndRotateIfTrue(password)
			if err != nil {
				log.Warn().Err(err).Str("Incomming", password).Str("Agent.Name", agentName).Str("Agent.ID", agent.Id).Msg("Failed to validate agent password")
				return false
			}
			err = db.UpdateAgentField(agent, "OneTimePassword")
			if err != nil {
				log.Warn().Err(err).Str("Agent.Name", agentName).Str("Agent.ID", agent.Id).Msg("Failed to update agent password")
				return false
			}
			fmt.Println("PASSWORD ACCEPTED")
			return true
		},
		// SessionRequestCallback logs information when a user requests a session
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
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

// handlePing returns pong when an agent send a ping
// This ping pong mechanism is used to perform a keepalive of the connections
func handlePing(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	// log.Trace().Msgf("SSH ping from %s", ctx.User())
	// log.Trace().Msgf("Returning pong to %s", ctx.User())
	log.Trace().Str("User", ctx.User()).Msg("PING received")
	return true, []byte("pong")
}
