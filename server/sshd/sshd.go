package sshd

import (
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"
	"Goauld/server/config"
	"Goauld/server/db"
	"context"
	"github.com/gliderlabs/ssh"
	"net"
)

func StartSshd(context context.Context, db *db.DB) {
	listener, err := net.Listen("tcp", ":61160")
	if err != nil {
		panic(err)
	}
	config.Get().SshPort = listener.Addr().(*net.TCPAddr).Port
	log.Debug().Msgf("SSH server listening on port %d", config.Get().SshPort)
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Addr:    "127.0.0.1:0",
		Banner:  "SSH-2.0-OpenSSH",
		Version: "SSH-2.0-OpenSSH",
		Handler: ssh.Handler(func(s ssh.Session) {
			srcAddr := s.RemoteAddr().String()
			s.User()
			log.Debug().Msgf("New ssh connection from %s (%s)", s.User(), srcAddr)
		}),
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Trace().Msgf("Reverse port forward for %s to %s", ctx.User(), host)
			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			// "direct-tcpip": ssh.DirectTCPIPHandler,
			"session": ssh.DefaultSessionHandler,
		},
		PublicKeyHandler: func(ctx ssh.Context, key ssh.PublicKey) bool {
			id := ctx.User()
			log.Debug().Msgf("SSH Connection attempt from %s", id)
			agent, err := db.FindAgent(id)
			if err != nil {
				log.Debug().Msgf("Agent not found (%s)", id)
				return false
			}

			agentPubKey, err := _ssh.ParseSSHPublicKey(agent.PublicKey)
			if err != nil {
				log.Debug().Msgf("Error parsing public key (%s)", id)
			}
			if ssh.KeysEqual(agentPubKey, key) {
				log.Trace().Msgf("SSH connection succeeded from %s (%s)", ctx.User(), id)
				return true
			}
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
				return true
			}
			return false
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			id := sess.User()
			log.Debug().Msgf("SSH session requested from %s", id)
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
