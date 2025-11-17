//go:build !mini

package control

import (
	"Goauld/agent/config"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	"errors"
	"fmt"
	"strings"

	sio "github.com/hazegard/socket.io-go"
	eio "github.com/hazegard/socket.io-go/engine.io"
	"github.com/coder/websocket"
	"github.com/xtaci/smux"
)

func GetRelayEioConfig(mode string, dnsTransport *transport.DNSSH) (*sio.ManagerConfig, error) {
	switch strings.ToLower(mode) {
	case "websocket":
		return getEioConfig([]string{"websocket"}), nil
	case "upgrade":
		return getEioConfig([]string{"polling", "websocket"}), nil
	case "polling":
		return getEioConfig([]string{"polling"}), nil
	case "dns":
		stream, err := dnsTransport.OpenStream()
		if err != nil {
			log.Error().Err(err).Msg("Error opening stream")

			return nil, err
		}

		_, err = stream.Write([]byte(config.Get().ID))
		if err != nil {
			log.Error().Err(err).Msg("Error writing to stream")

			return nil, err
		}
		_, err = stream.Write([]byte{'C'})
		if err != nil {
			return nil, fmt.Errorf("error writing id to DNS tunnelled session: %w", err)
		}

		return getDNSEioConfig(stream), nil
	}

	return nil, errors.New("unable to open Relay Socket.IO config")
}

// getEioConfig return the socket.io underlying configuration.
func getDNSEioConfig(session *smux.Stream) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Str("Transport", transportName).Msg("Client transport upgrade done")
			},
			HTTPTransport: NewSmuxTransport(session),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: newSmuxHTTPandHTTPSClient(session),
			},
			// When tunneling over DNS, if we use polling only or polling then websocket upgrade,
			// The tunnel fails to establish properly as the server responds to unwanted content to the open HTTP socket.
			// Here we use the full duplex websocket mechanism to ensure that the tunnel is properly working
			// On the client side
			Transports: []string{"websocket"},
		},
	}
}

// InitOverDNS tries to connect to the control plan using the DNS transport.
func (cpc *ControlPlanClient) InitOverDNS(session *smux.Stream, success chan<- struct{}, chanErr chan<- error) error {
	_, err := session.Write([]byte(config.Get().ID))
	// DNS MODE means we are using http to simplify the exchanges
	u := strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoURL(config.Get().ID), "https://"), "http://")
	cpc.url = "http://" + u
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %w", err)
	}
	_, err = session.Write([]byte{'C'})
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %w", err)
	}
	cfg := getDNSEioConfig(session)

	return cpc.init(cfg, success, chanErr)
}
