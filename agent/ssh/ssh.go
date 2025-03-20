package ssh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"Goauld/agent/config"
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"

	"golang.org/x/crypto/ssh"
)

type SSHAgent struct {
	client *ssh.Client
	ctx    context.Context
	conn   net.Conn

	remotePortMapMu sync.Mutex
	remotePortMap   map[string]_ssh.RemotePortForwarding
}

// NewSSHAgent returns a new Agent
func NewSSHAgent() *SSHAgent {
	return &SSHAgent{
		remotePortMap: make(map[string]_ssh.RemotePortForwarding),
	}
}

// Init initialize the ssh client using the configuration
func (sshAgent *SSHAgent) Init(ctx context.Context) error {
	log.Info().Msg("Connecting to the ssh server...")
	// Get the private key used to authenticate to the server
	privateKey, err := ssh.ParsePrivateKey([]byte(config.Get().SShPrivateKey))
	if err != nil {
		return err
	}
	sshConfig := &ssh.ClientConfig{
		User:            config.Get().Id,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(privateKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		ClientVersion:   "SSH-2.0-Client",
	}

	// defer cancel()
	// Get the ssh client, which may be proxified to bypass proxies
	client, conn, err := getProxifiedClient(sshConfig, ctx)
	if err != nil {
		log.Error().Err(err).Msg("ssh init client failed")
		return err
	}

	// start a keepalive loop
	go sshAgent.sshKeepAliveLoop(ctx)

	sshAgent.client = client
	sshAgent.ctx = ctx
	sshAgent.conn = conn

	go func() {
		select {
		case <-ctx.Done():
			err := sshAgent.Close()
			if err != nil {
				log.Error().Err(err).Msg("ssh client close failed")
				return
			}
		}
	}()

	return nil
}

// GetRemoteConn returns a net.Listener listening on the ssh server host, as well as the port used by the remote listener
func (sshAgent *SSHAgent) GetRemoteConn(remote string) (net.Listener, int, error) {
	l, err := sshAgent.client.Listen("tcp", remote)
	if err != nil {
		return nil, 0, err
	}
	port := l.Addr().(*net.TCPAddr).Port
	return l, port, err
}

// RemoteForward starts the remote port forwarded in background. It returns the remote listening port
func (sshAgent *SSHAgent) RemoteForward(rpf _ssh.RemotePortForwarding, ctx context.Context) (int, error) {
	// start the remote forwarding to remotely expose the local sshd server
	remoteListener, err := sshAgent.client.Listen("tcp", rpf.GetRemote())
	if err != nil {
		return 0, fmt.Errorf("failed to start remote listener: %w", err)
	}

	remotePort := remoteListener.Addr().(*net.TCPAddr).Port
	rpf.ServerPort = remotePort

	sshAgent.remotePortMapMu.Lock()
	sshAgent.remotePortMap[rpf.String()] = rpf
	sshAgent.remotePortMapMu.Unlock()

	// Loop that perform forwarding from the remote connection
	// to the local sshd server
	go func() {
		defer func() {
			sshAgent.remotePortMapMu.Lock()
			delete(sshAgent.remotePortMap, rpf.String())
			sshAgent.remotePortMapMu.Unlock()
		}()

		defer remoteListener.Close()
		for {
			if ctx.Err() != nil {
				remoteListener.Close()
				return
			}

			// Waits for a connection
			remoteConn, err := remoteListener.Accept()
			if err != nil {
				// TODO faire du throttle si on garde l'erreur, voir pour couper proprement après un temp ?
				log.Error().Err(err).Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Msg("failed to accept remote connection")
				// Pseudo throttle en attendant
				time.Sleep(1 * time.Second)
				continue
			}

			// Handle the connection in a dedicated goroutine
			go func() {
				// Initialize a connection to the local SSHD server
				localConn, err := net.Dial("tcp", rpf.GetLocal())
				if err != nil {
					log.Error().Err(err).Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Msg("failed to connect to local service")
					return
				}
				defer localConn.Close()

				errChan := make(chan error, 1)

				// Initialize the Websocket -> SSH connection
				go func() {
					_, err := io.Copy(localConn, remoteConn)
					if err != nil && !errors.Is(err, io.EOF) {
						log.Error().Err(err).Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Msgf("Remote forwarding: Local -> Remote connection failed")
						errChan <- err
					}
				}()

				// Initialize the SSH -> Websocket connection
				go func() {
					_, err := io.Copy(remoteConn, localConn)
					if err != nil && !errors.Is(err, io.EOF) {
						log.Error().Err(err).Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Msgf("Remote forwarding: Remote -> Local connection failed")
						errChan <- err
					}
				}()

				// Waits for an error to occur
				err = <-errChan
				remoteConn.Close()
				localConn.Close()
				if err != nil {
					log.Error().Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Err(err).Msg("Remote forwarding: end of forwarding")
				}
			}()
		}
	}()
	return remotePort, nil
}

// sshKeepAliveLoop starts a loop that will periodically send ping messages to the server
// in order to perform a keepalive to ensure that the connection is kept active even if no traffic
// is transmitted within the connection
func (sshAgent *SSHAgent) sshKeepAliveLoop(ctx context.Context) {
	t := time.NewTicker(config.Get().GetKeepalive() * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Trace().Msg("Sending keepalive to SSH connection")
			_, b, err := sshAgent.client.SendRequest("ping", true, nil)
			if err != nil {
				return
			}
			log.Trace().Msgf("Keepalive %s recveived from server", string(b))
		case <-ctx.Done():
			return
		}
	}
}

// Close closes the ssh connection
func (sshAgent *SSHAgent) Close() error {
	log.Warn().Msg("Shutting down SSH agent...")
	if sshAgent.conn != nil {
		sshAgent.conn.Close()
	}
	return sshAgent.client.Close()
}
