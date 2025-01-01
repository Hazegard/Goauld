package ssh

import (
	"Goauld/agent/agent"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"context"
	"fmt"
	"golang.org/x/crypto/ssh"
	"net"
	"strings"
	"time"
)

// getProxifiedClient return a connected SSH client
// This client may be proxyfies (TLS, Websocket, HTTP), or not, depending
// on the egress restrictions
// The order of the connection attempt is defined in the agent configuration
func getProxifiedClient(sshConfig *ssh.ClientConfig, ctx context.Context) (*ssh.Client, error) {
	var client *ssh.Client
	for _, proto := range agent.Get().GetRsshOrder() {
		switch {
		case strings.HasPrefix(proto, "ssh"):
			client = directSSH(sshConfig)
			if client != nil {
				return client, nil
			}

		case strings.HasPrefix(proto, "ws"):
			client = proxifyWS(sshConfig, ctx)
			if client != nil {
				return client, nil
			}
		case strings.HasPrefix(proto, "http"):
			client = proxifyHttp(sshConfig)
			if client != nil {
				return client, nil
			}
		case strings.HasPrefix(proto, "tls"):
			client = proxifyTls(sshConfig, ctx)
			if client != nil {
				return client, nil
			}
		}
	}

	return nil, fmt.Errorf("failed to Proxify ssh connection")
}

// directSSH perform a direct ssh connection to the SSHD server
func directSSH(sshConfig *ssh.ClientConfig) *ssh.Client {
	log.Info().Msgf("Trying to direct connect to ssh server")
	client, err := transport.DirectSshConnect(sshConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect directly to ssh server")
		return nil
	}
	log.Info().Msgf("Direct connection to the ssh server successfully")
	return client
}

// proxifyTls proxifies the SSH traffic using a TLS connection to the server
func proxifyTls(sshConfig *ssh.ClientConfig, ctx context.Context) *ssh.Client {
	log.Info().Msgf("Trying to proxify SSH using TLS")
	tlsConn, err := transport.GetTlsConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create TLS connection")
		return nil
	}
	log.Debug().Msg("Connection succedded, trying to mount SSH over the TLS connection")
	client, err := tryProxifySsh(sshConfig, tlsConn)
	if err != nil {
		log.Error().Err(err).Msg("Failed to proxify SSH over the TLS connection")
		return nil
	}
	log.Info().Msg("Proxify using TLS succeeded")
	return client
}

// proxifyTls proxifies the SSH traffic using a websocket connection to the server
func proxifyWS(sshConfig *ssh.ClientConfig, ctx context.Context) *ssh.Client {
	log.Info().Msg("Trying to proxify SSH using websocket")
	wsConn, err := transport.GetWebsocketConn(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create WebSocket connection")
		return nil
	}
	log.Debug().Msg("Connection succedded, trying to mount SSH over the Websocket connection")
	client, err := tryProxifySsh(sshConfig, wsConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to proxify ssh connection using websocket")
		return nil
	}
	log.Info().Msg("Proxify using websocket succeeded")
	return client
}

// proxifyTls proxifies the SSH traffic using a HTTP connection to the server
func proxifyHttp(sshConfig *ssh.ClientConfig) *ssh.Client {
	log.Info().Msg("Trying to proxify SSH using HTTP")
	httpConn := transport.NewSSHTTPConn()
	err := httpConn.Connect()
	if err != nil {
		log.Error().Err(err).Msg("failed to proxify SSH using HTTP")
		return nil
	}
	log.Debug().Msg("Connection succedded, trying to mount SSH over the HTTP connection")
	//httpConn.Start()
	client, err := tryProxifySsh(sshConfig, httpConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to proxify ssh connection using HTTP")
		return nil
	}
	log.Info().Msg("Proxify using HTTP succeeded")
	return client
}

// tryProxifySsh attempts to proxifies the SSH connection using the provided net.Conn
// A 30 seconds timeout is used if the underlying connection hangs without being fully established
// or without failure
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
