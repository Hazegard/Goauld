package sshd

import (
	"Goauld/agent/clipboard"
	"Goauld/agent/config"
	globalcontext "Goauld/agent/context"
	"Goauld/agent/sshd/shell"
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"
	"context"
	"errors"
	"io"
	"net"
	"strings"

	"github.com/charmbracelet/ssh"
	gossh "golang.org/x/crypto/ssh"
)

// Sshd holds the SSHD server.
type Sshd struct {
	server   *ssh.Server
	listener net.Listener
}

// NewSshdServer configure and return an SSHD server.
func NewSshdServer(ctx context.Context, canceler *globalcontext.GlobalCanceler) *Sshd {
	forwardHandler := &ssh.ForwardedTCPHandler{}
	s := &ssh.Server{
		Handler: ssh.Handler(func(s ssh.Session) {
			log.Info().Str("Username", s.User()).Str("RemoteAddr", s.RemoteAddr().String()).Str("Command", s.RawCommand()).Msgf("New connection from %s with username %s", s.RemoteAddr(), s.User())
			if strings.Contains(s.RawCommand(), "rsync --server") || (len(s.Command()) > 0 && s.Command()[0] == "rsync") {
				HandleRsync(s)

				return
			}
			err := shell.GivePty(ctx, s, s.Command(), s.RawCommand())
			if err != nil {
				log.Error().Err(err).Msg("error spawning pty")
			}
			_ = s.Close()
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
			_ssh.Copy: func(ctx ssh.Context, _ *ssh.Server, _ *gossh.Request) (bool, []byte) {
				log.Trace().Str("Event", _ssh.Copy).Msgf("Received COPY from %s", ctx.User())
				content, err := clipboard.Copy(ctx)
				if err != nil {
					log.Error().Err(err).Msg("clipboard copy")

					return false, content
				}

				return true, content
			},
			_ssh.Paste: func(ctx ssh.Context, _ *ssh.Server, req *gossh.Request) (bool, []byte) {
				log.Trace().Str("Event", _ssh.Paste).Str("Content", string(req.Payload)).Msgf("Received PASTE from %s", ctx.User())
				err := clipboard.Paste(ctx, req.Payload)
				if err != nil {
					log.Error().Err(err).Msg("clipboard copy")

					return false, nil
				}

				return true, nil
			},
			_ssh.Kill: func(ctx ssh.Context, _ *ssh.Server, req *gossh.Request) (bool, []byte) {
				log.Trace().Str("Event", _ssh.Kill).Str("Content", string(req.Payload)).Msgf("Received KILL from %s", ctx.User())
				canceler.Exit("Client requested exit")

				return true, nil
			},
			_ssh.Restart: func(ctx ssh.Context, _ *ssh.Server, req *gossh.Request) (bool, []byte) {
				log.Trace().Str("Event", _ssh.Restart).Str("Content", string(req.Payload)).Msgf("Received RESTART from %s", ctx.User())
				canceler.Exit("Client requested restart")

				return true, nil
			},
			_ssh.Delete: func(ctx ssh.Context, _ *ssh.Server, req *gossh.Request) (bool, []byte) {
				log.Trace().Str("Event", _ssh.Delete).Str("Content", string(req.Payload)).Msgf("Received DELETE from %s", ctx.User())
				canceler.Delete("Client requested restart")

				return true, nil
			},
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
		PtyCallback: func(ctx ssh.Context, _ ssh.Pty) bool {
			log.Trace().Msgf("Received pty request from user: %s", ctx.User())

			return true
		},
		SessionRequestCallback: func(sess ssh.Session, _ string) bool {
			log.Trace().Msgf("Received session request from user: %s", sess.User())

			return true
		},
		SubsystemHandlers: map[string]ssh.SubsystemHandler{
			"sftp":  SftpHandler,
			"rsync": HandleRsync,
		},
	}
	// This is an attempt to use builtin charmbracelet/ssh pty
	// Without success (see agent/sshd/shell/shell.go)
	// s.SetOption(ssh.AllocatePty())

	return &Sshd{server: s}
}

// Serve serves the sshd server.
func (sshd *Sshd) Serve(l net.Listener) error {
	sshd.listener = l
	err := sshd.server.Serve(l)
	if errors.Is(err, io.EOF) {
		return nil
	}

	return err
}

// Close closes the server and is underlying listeners.
func (sshd *Sshd) Close() error {
	log.Warn().Msg("Shutting done the SSHD server")
	err := sshd.listener.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		log.Error().Err(err).Msg("Error closing SSHD listener")
	}

	return sshd.server.Close()
}
