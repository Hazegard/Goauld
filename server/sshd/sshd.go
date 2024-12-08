package sshd

import (
	"Goauld/client/sshd/shell"
	"Goauld/server/structs"
	"fmt"
	"github.com/gliderlabs/ssh"
	"log"
	"net"
)

var store structs.AgentStore

func StartSShd() {
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
			fmt.Println("New connection from ", key)
			store.Get(key)
			err = shell.GivePty(s, s.Command())
			if err != nil {
				log.Printf("error spawning pty: %s\n", err)
			}
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
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			fmt.Println(password)
			key := ctx.RemoteAddr().String()
			agent := store.Get(key)
			if password == agent.Password {
				return true
			}
			return false
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			fmt.Printf("session requested: %s\n", requestType)
			return true
		},
	}
	fmt.Println(s.Serve(listener))
	fmt.Println("Shutting down...")
}
