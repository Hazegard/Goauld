package sshd

import (
	_ssh "Goauld/common/ssh"
	"Goauld/server/db"
	"context"
	"fmt"
	"github.com/gliderlabs/ssh"
	"log"
	"net"
)

func StartSshd(context context.Context) {
	listener, err := net.Listen("tcp", ":61160")
	if err != nil {
		panic(err)
	}
	fmt.Println("Listening on port ", listener.Addr().(*net.TCPAddr).Port)
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Addr: "127.0.0.1:0",
		Handler: ssh.Handler(func(s ssh.Session) {
			key := s.RemoteAddr().String()
			s.User()
			fmt.Println("New connection from ", key)
			// err = shell.GivePty(s, s.Command())
			// if err != nil {
			// 	log.Printf("error spawning pty: %s\n", err)
			// }
		}),
		// LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
		// 	fmt.Println("Forwarding to", destinationHost)
		// 	return true
		// },
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			fmt.Println("Forwarding from", host)
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
			fmt.Println("PublicKey from", ctx.User())
			user := ctx.User()
			agent, err := db.Get().FindAgent(user)
			if err != nil {
				fmt.Println(err)
				return false
			}

			agentPubKey, err := _ssh.ParseSSHPublicKey(agent.PublicKey)
			if err != nil {
				fmt.Println(err)
			}
			if ssh.KeysEqual(agentPubKey, key) {
				return true
			}
			return false
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			remote := ctx.RemoteAddr().String()
			id := ctx.User()
			agent, err := db.Get().FindAgent(id)
			if err != nil {
				log.Printf("error getting agent from store: %s\n", err)
				return false
			}
			agent.Source = remote
			// TODO: ici c'est pas le mdp de l'agent mais le mdp du serveur à revoir
			if password == "" {
				return true
			}
			return false
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			fmt.Printf("session requested: %s\n", requestType)
			return false
		},
	}
	fmt.Println(s.Serve(listener))
	fmt.Println("Shutting down...")
	context.Done()
}
