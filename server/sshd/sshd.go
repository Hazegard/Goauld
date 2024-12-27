package sshd

import (
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"
	"Goauld/server/config"
	"Goauld/server/db"
	"context"
	"github.com/gliderlabs/ssh"
	"net"

	gossh "golang.org/x/crypto/ssh"
)

func StartSshd(context context.Context, db *db.DB) {
	listener, err := net.Listen("tcp", config.Get().LocalSShServer())
	if err != nil {
		panic(err)
	}
	// Update if listening on 0 to get the real port
	config.Get().SshPort = listener.Addr().(*net.TCPAddr).Port
	log.Debug().Msgf("SSH server listening on port %s", config.Get().LocalSShServer())
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Addr:    "127.0.0.1:0",
		Version: "Server",
		Handler: ssh.Handler(func(s ssh.Session) {
			srcAddr := s.RemoteAddr().String()
			s.User()
			log.Debug().Str("User", s.User()).Msgf("New ssh connection from %s", srcAddr)
		}),
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Trace().Str("User", ctx.User()).Msgf("Reverse port forward to %s", host)
			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
			"ping":                 handlePing,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			// "direct-tcpip": ssh.DirectTCPIPHandler,
			"session": ssh.DefaultSessionHandler,
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			id := ctx.User()
			remote := ctx.RemoteAddr().String()
			log.Debug().Str("User", id).Str("Remote", remote).Msgf("SSH Connection attempt")
			agent, err := db.FindAgent(id)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", id)
				return false
			}
			log.Trace().Str("User", id).Msg("Agent found, getting public key...")

			agentPubKey, err := _ssh.ParseSSHPublicKey(agent.PublicKey)
			if err != nil {
				log.Debug().Msgf("Error parsing public key (%s)", id)
			}
			log.Trace().Str("User", id).Msg("Public Key found, checking public key...")
			if ssh.KeysEqual(agentPubKey, key) {
				log.Trace().Msgf("SSH connection succeeded from %s (%s)", ctx.User(), id)
				return true
			}
			log.Warn().Str("User", id).Str("Remote", remote).Msg("Wrong Public Key...")
			return false
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			remote := ctx.RemoteAddr().String()
			id := ctx.User()
			agent, err := db.FindAgent(id)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", id)
				return false
			}
			agent.Source = remote
			// TODO: ici c'est pas le mdp de l'agent mais le mdp du serveur à revoir
			if password == agent.SharedSecret {
				return false
			}
			return false
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			id := sess.User()
			remote := sess.RemoteAddr().String()
			log.Trace().Str("User", id).Str("Remote", remote).Msgf("SSH session requested from %s", id)
			return false
		},
	}
	err = s.Serve(listener)
	if err != nil {
		log.Error().Msg(err.Error())
	}
	log.Println("SSH server stopped")
	context.Done()
}

func handlePing(ctx ssh.Context, srv *ssh.Server, req *gossh.Request) (bool, []byte) {
	// log.Trace().Msgf("SSH ping from %s", ctx.User())
	// log.Trace().Msgf("Returning pong to %s", ctx.User())
	log.Trace().Str("User", ctx.User()).Msg("PING received")
	return true, []byte("pong")
}
