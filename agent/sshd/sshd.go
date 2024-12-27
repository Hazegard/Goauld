package sshd

import (
	"Goauld/agent/agent"
	"Goauld/agent/sshd/shell"
	"Goauld/common/log"
	"github.com/gliderlabs/ssh"
	"net"
)

func StartSShd() error {
	sshdAddress := agent.Get().LocalSShdAddress()
	listener, err := net.Listen("tcp", sshdAddress)
	if err != nil {
		return err
	}
	if agent.Get().IsLocalSshdRandomPort() {
		agent.Get().SetLocalSshdPort(listener.Addr().(*net.TCPAddr).Port)
	}

	log.Info().Msg("start sshd")
	log.Info().Msgf("Listening on port %s", agent.Get().LocalSShdAddress())
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Handler: ssh.Handler(func(s ssh.Session) {
			log.Info().Msg("sshd")
			log.Debug().Msgf("New connection from %s with username %s", s.RemoteAddr(), s.User())
			err = shell.GivePty(s, s.Command())
			if err != nil {
				log.Error().Err(err).Msg("error spawning pty")
			}

		}),
		LocalPortForwardingCallback: func(ctx ssh.Context, destinationHost string, destinationPort uint32) bool {
			log.Trace().Msgf("Forwarding connection from %s to %s:%d", ctx.User(), destinationHost, destinationPort)
			return true
		},
		ReversePortForwardingCallback: func(ctx ssh.Context, host string, port uint32) bool {
			log.Trace().Msgf("Forwarding connection to %s from %s:%d", ctx.User(), host, port)
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
			log.Debug().Msgf("Received connection from user: %s", ctx.User())
			if password == agent.Get().LocalSShdPassword() {
				log.Debug().Msgf("Connnection using password succecced from user: %s", ctx.User())
				return true
			}
			log.Debug().Msgf("Connnection using password failed from user: %s", ctx.User())
			return false
		},
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
	return s.Serve(listener)
}
