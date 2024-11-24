package sshd

import (
	"Goauld/sshd/shell"
	"fmt"
	"github.com/gliderlabs/ssh"
	"net"
	"runtime"
)

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
			fmt.Println("New connection from ", s.RemoteAddr())

			switch runtime.GOOS {
			case "darwin":

				fmt.Println(s.Command())
				shell.GivePty(s, s.Command())
			case "linux":
			case "windows":
			default:
				fmt.Println("Unsupported platform")
				return
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
			if password == "password" {
				return true
			}
			return true
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
	fmt.Println(s.Serve(listener))
	fmt.Println("Shutting down...")
}
