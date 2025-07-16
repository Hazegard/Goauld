package sshd

import (
	"context"
	"io"
	"net"

	"Goauld/agent/config"
	"Goauld/agent/sshd/shell"
	"Goauld/common/log"
	"github.com/gliderlabs/ssh"
)

type Sshd struct {
	server   *ssh.Server
	listener net.Listener
}

// NewSshdServer configure and return an SSHD server
func NewSshdServer(ctx context.Context) *Sshd {
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Handler: ssh.Handler(func(s ssh.Session) {
			log.Info().Msgf("New connection from %s with username %s", s.RemoteAddr(), s.User())
			err := shell.GivePty(s, s.Command(), ctx)
			if err != nil {
				log.Error().Err(err).Msg("error spawning pty")
			}
			s.Close()
		}),
		// Allows Local Port Forwarding
		// Note: this might be unnecessary as users should only perform reverse port forwarding
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			log.Trace().Msgf("Forwarding connection from %s to %s:%d", ctx.User(), destinationHost, destinationPort)
			return true
		},
		// Allows Reverse Port Forwarding
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Trace().Msgf("Forwarding connection to %s from %s:%d", ctx.User(), host, port)
			return true
		},
		// Allows tcp traffic within the SSH connection
		RequestHandlers: map[string]ssh.RequestHandler{
			"tcpip-forward":        forwardHandler.HandleSSHRequest,
			"cancel-tcpip-forward": forwardHandler.HandleSSHRequest,
		},
		// Allows channels
		ChannelHandlers: map[string]ssh.ChannelHandler{
			"direct-tcpip": ssh.DirectTCPIPHandler,
			"session":      ssh.DefaultSessionHandler,
		},
		// Allows tcp traffic within the SSH connection
		PasswordHandler: func(ctx ssh.Context, password string) bool {
			log.Debug().Msgf("Received connection from user: %s", ctx.User())
			if config.Get().ValidatePassword(password) {
				log.Debug().Msgf("Connnection using password succecced from user: %s", ctx.User())
				return true
			}
			log.Debug().Msgf("Connnection using password failed from user: %s", ctx.User())
			return false
		},
		// Allows opening a shell
		PtyCallback: func(ctx ssh.Context, pty ssh.Pty) bool {
			log.Trace().Msgf("Received pty request from user: %s", ctx.User())
			return true
		},
		SessionRequestCallback: func(sess ssh.Session, requestType string) bool {
			log.Trace().Msgf("Received session request from user: %s", sess.User())
			return true
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp": SftpHandler,
		},
	}
	return &Sshd{server: s}
}

func (sshd *Sshd) Serve(l net.Listener) error {
	sshd.listener = l
	err := sshd.server.Serve(l)
	if err == io.EOF {
		return nil
	}
	return err
}

func (sshd *Sshd) Close() error {
	log.Warn().Msg("Shutting done the SSHD server")
	err := sshd.listener.Close()
	if err != nil && err != io.EOF {
		log.Error().Err(err).Msg("Error closing SSHD listener")
	}
	return sshd.server.Close()
}
