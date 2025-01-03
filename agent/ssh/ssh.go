package ssh

import (
	"Goauld/agent/agent"
	"Goauld/common/log"
	_ssh "Goauld/common/ssh"
	"context"
	"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"sync"
	"time"
)

type SSHAgent struct {
	client *ssh.Client
	ctx    context.Context

	remotePortMapMu sync.Mutex
	remotePortMap   map[string]_ssh.RemotePortForwarding
}

func NewSSHAgent() *SSHAgent {
	return &SSHAgent{
		remotePortMap: make(map[string]_ssh.RemotePortForwarding),
	}
}

func (sshAgent *SSHAgent) Init() error {
	log.Info().Msg("Connecting to the ssh server...")
	// Get the private key used to authenticate to the server
	privateKey, err := ssh.ParsePrivateKey([]byte(agent.Get().SShPrivateKey))
	if err != nil {
		return err
	}
	sshConfig := &ssh.ClientConfig{
		User:            agent.Get().Id,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(privateKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		ClientVersion:   "SSH-2.0-Client",
	}

	ctx := context.Background()
	// defer cancel()
	// Get the ssh client, which may be proxified to bypass proxies
	client, err := getProxifiedClient(sshConfig, ctx)
	if err != nil {
		log.Error().Err(err).Msg("ssh init client failed")
		return err
	}

	// start a keepalive loop
	go sshAgent.sshKeepAliveLoop()

	sshAgent.client = client
	sshAgent.ctx = ctx
	return nil
}

func (sshAgent *SSHAgent) GetRemoteConn(remote string) (net.Listener, int, error) {
	l, err := sshAgent.client.Listen("tcp", remote)
	if err != nil {
		return nil, 0, nil
	}
	port := l.Addr().(*net.TCPAddr).Port
	return l, port, err
}

// RemoteForward starts the
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
				return
			}

			// Waits for a connection
			remoteConn, err := remoteListener.Accept()
			if err != nil {
				// TODO faire du throttle si on garde l'erreur, voir pour cuoper proprement après un temp ?
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
				if err != nil {
					log.Error().Str("Local", rpf.GetLocal()).Str("Remote", rpf.GetRemote()).Err(err).Msg("Remote forwarding: end of forwarding")
				}
				sshAgent.Close()
			}()
		}
	}()
	return remotePort, nil
}

// sshKeepAliveLoop starts a loop that will periodically send ping messages to the server
// in order to perform a keepalive to ensure that the connection is kept active even if no traffic
// is transmitted within the connection
func (sshAgent *SSHAgent) sshKeepAliveLoop() {
	t := time.NewTicker(agent.Get().GetKeepalive() * time.Second)
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
		case <-sshAgent.ctx.Done():
			return
		}

	}
}

func (sshAgent *SSHAgent) Close() error {
	sshAgent.ctx.Done()
	return sshAgent.client.Close()
}
