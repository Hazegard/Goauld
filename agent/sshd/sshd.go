package sshd

import (
	"Goauld/agent/agent"
	"Goauld/agent/sshd/shell"
	"fmt"
	"github.com/gliderlabs/ssh"
	"log"
	"net"
)

func StartSShd() error {
	sshdAddress := agent.Get().LocalSShdAddress()
	listener, err := net.Listen("tcp", sshdAddress)
	if err != nil {
		return err
	}
	if agent.Get().IsSshdRandomPort() {
		agent.Get().SetLocalSshdPort(listener.Addr().(*net.TCPAddr).Port)
	}

	fmt.Println("Listening on port ", agent.Get().LocalSShdAddress())
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Handler: ssh.Handler(func(s ssh.Session) {
			fmt.Println("New connection from ", s.RemoteAddr())
			err = shell.GivePty(s, s.Command())
			if err != nil {
				log.Printf("error spawning pty: %s\n", err)
			}

		}),
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			fmt.Println("Forwarding to", destinationHost)
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			fmt.Println("Forwarding from", host)
			return true
		},
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
		},
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			fmt.Println(password)
			fmt.Println(agent.Get().LocalSShdPassword())
			if password == agent.Get().LocalSShdPassword() {
				return true
			}
			return false
		},
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			fmt.Println("pty requested")
			return true
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			fmt.Printf("session requested: %s\n", requestType)
			return true
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
	}
	return s.Serve(listener)
}
