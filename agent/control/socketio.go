package control

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/quic-go/webtransport-go"

	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"Goauld/common/crypto"
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"Goauld/common/ssh"
	sio "github.com/karagenc/socket.io-go"
	eio "github.com/karagenc/socket.io-go/engine.io"
	"nhooyr.io/websocket"
)

// ControlPlanClient Handle the socket.io interaction regarding the management of the agent
type ControlPlanClient struct {
	manager    *sio.Manager
	socket     sio.ClientSocket
	configDone chan<- struct{}
	ctx        context.Context
	url        string
	cancel     context.CancelFunc
}

// NewControlPlanClient returns a new ControlPlanClient
func NewControlPlanClient(ctx context.Context, configDone chan<- struct{}, cancel context.CancelFunc) *ControlPlanClient {
	return &ControlPlanClient{
		ctx:        ctx,
		url:        config.Get().SocketIoUrl(),
		configDone: configDone,
		cancel:     cancel,
	}
}

// Init initialize the socket.io handlers
func (cpc *ControlPlanClient) Init() error {
	cfg := getEioConfig()
	manager := sio.NewManager(cpc.url, cfg)
	socket := manager.Socket("/", nil)

	socket.OnConnect(func() {
		log.Trace().Msg("OnConnect")
		log.Info().Msgf("Connected to the control server %s", cpc.url)
	})
	socket.OnConnectError(func(err any) {
		log.Trace().Msg("OnConnectError")
		log.Error().Msgf("Error occured connecting to %s (%v)", cpc.url, err)
	})

	manager.OnError(func(err error) {
		log.Trace().Msg("OnError")
		log.Error().Err(err).Msgf("Error occured  %s", cpc.url)
	})
	manager.OnReconnect(func(attempt uint32) {
		log.Trace().Msg("OnReconnect")
		log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", cpc.url, attempt)
	})

	// SendSshPrivateKeyEvent is sent by the server after the client sends the RegisterEvent event
	// this event contains the encrypted SSH private key used by the agent to authenticate on the
	// SSHD server.
	// Once received, the agent sends its SSHD password to the server using the SendAgentDataEvent event
	socket.OnEvent(socketio.SendSshPrivateKeyEvent, func(data []byte) {
		log.Trace().Msg("OnEvent: SendSshPrivateKeyEvent")
		log.Trace().Msgf("SshPrivateKeyEvent: data reveived")
		// Decrypt the SSH private key
		privateKey, err := socketio.DecryptSshPrivateKeyMessage(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error decrypting private key")
		}

		// Add the decrypted SSH private key to the agent configuration
		config.Get().SShPrivateKey = privateKey.SshPrivateKey
		log.Debug().Msgf("Ssh private key received and successfully decrypted")
		log.Debug().Msgf("Sending local sshd password")
		// Encrypt the SSH password used by the client to authenticate to the agent SSHD server
		localSshPassword, err := socketio.NewEncryptedAgentSshPasswordMessage(config.Get(), config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error encrypting local sshd password")
		}
		log.Debug().Msgf("Local sshd password sent")
		// Send the encrypted SSH password to the server
		socket.Emit(socketio.SendAgentDataEvent, localSshPassword)

		log.Trace().Msg("OnEvent: SendSshPrivateKeyEvent done")
	})

	// SendSshHPrivateKeyError Logs when the server returns an error
	socket.OnEvent(socketio.SendSshHPrivateKeyError, func() {
		log.Trace().Msg("OnEvent: SendSshHPrivateKeyError")
		log.Error().Msgf("Error occured (%s) %s", "SendSshHPrivateKeyError", cpc.url)
		log.Trace().Msg("OnEvent: SendSshHPrivateKeyError done")
	})

	// SendSshPrivateKeySuccess Logs when the server returns no error
	socket.OnEvent(socketio.SendSshPrivateKeySuccess, func() {
		log.Trace().Msg("OnEvent: SendSshPrivateKeySuccess")
		log.Debug().Msgf("Event SendSshPrivateKeySuccess received")
		log.Trace().Msg("OnEvent: SendSshPrivateKeySuccess done")
	})

	// SendAgentDataError Logs when the server returns an error
	socket.OnEvent(socketio.SendAgentDataError, func() {
		log.Trace().Msg("OnEvent: SendAgentDataError")
		log.Error().Msgf("Error occured (%s) %s", "SendAgentDataError", cpc.url)
		log.Trace().Msg("OnEvent: SendAgentDataError done")
	})

	// SendAgentDataSuccess Logs when the server returns no error
	// As it complete the configuration steps between the
	socket.OnEvent(socketio.SendAgentDataSuccess, func() {
		log.Trace().Msg("OnEvent: SendAgentDataSuccess")
		cpc.configDone <- struct{}{}
		log.Trace().Msg("OnEvent: SendAgentDataSuccess done")
	})

	socket.OnEvent(socketio.RegisterError, func(data socketio.SioError) {
		log.Error().Err(errors.New(data.Message)).Msgf("Error occured %s", "RegisterError")
		log.Info().Msgf("Quitting...")
		socket.Disconnect()
		os.Exit(2)
	})

	socket.OnEvent(socketio.ExitEvent, func(doExit bool) {
		log.Info().Msg("OnEvent: Exit requested")
		socket.Emit(socketio.ExitSuccess)
		socket.Disconnect()
		if doExit {
			os.Exit(0)
		}
		cpc.cancel()
		cpc.Close()
	})

	socket.OnEvent(socketio.AlreadyConnectedEvent, func() {
		log.Info().Msg("AlreadyConnectedEvent: Exit requested because agent is already running")
		socket.Emit(socketio.ExitSuccess)
		socket.Disconnect()
		os.Exit(0)
	})

	socket.OnEvent(socketio.SendRemotePortForwardingDataSuccess, func() {
		log.Trace().Msg("OnEvent: SendRemotePortForwardingDataSuccess")
		log.Info().Msgf("SendRemotePortForwardingDataSuccess successfully sent")
		log.Trace().Msg("OnEvent: SendRemotePortForwardingDataSuccess done")
		log.OK().Msg("Agent successfully started.")
	})

	cpc.socket = socket
	cpc.manager = manager
	return nil
}

// Start starts the socket and initiates the configuration exchages with the server
func (cpc *ControlPlanClient) Start() error {
	encryptedKey, err := crypto.AsymEncrypt(config.Get().AgePubKey(), config.Get().SharedSecret)
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %v", err)
	}

	encryptedName, err := crypto.AsymEncrypt(config.Get().AgePubKey(), config.Get().Name())
	if err != nil {
		return fmt.Errorf("error encrypting shared secret: %v", err)
	}

	// This will be emitted after the socket is connected.
	cpc.socket.Emit(socketio.RegisterEvent, socketio.Register{
		Id:        config.Get().Id,
		SharedKey: encryptedKey,
		Name:      encryptedName,
	})

	cpc.socket.Connect()
	// starts the keepalive in background
	go cpc.keepAliveLoop(cpc.ctx)
	log.Debug().Msgf("Connected to the control server %s", cpc.url)
	log.Trace().Msg("Event send: RegisterEvent")
	// Waits for an error or the end of the socket
	select {
	case <-cpc.ctx.Done():
		log.Warn().Msgf("Shutting done the socketio control socket")
		cpc.socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
		log.Trace().Msg("Event send: Disconnect")
		cpc.socket.Disconnect()
	}
	return nil
}

// SendPorts sends the remote ports used by the agent
func (cpc *ControlPlanClient) SendPorts(rpf []ssh.RemotePortForwarding) error {
	data, err := socketio.EncryptRemotePortForwardingMessage(rpf, config.Get().Cryptor)
	if err != nil {
		return fmt.Errorf("error encrypting remote port forwarding message: %v", err)
	}
	cpc.socket.Emit(socketio.SendRemotePortForwardingDataEvent, data)
	return nil
}

// KeepAliveLoop starts a keepalive loop that will periodically send ping
// in order to keep alive the connection
func (cpc *ControlPlanClient) keepAliveLoop(ctx context.Context) {
	cpc.socket.OnEvent(socketio.PongEvent, func(data []byte) {
		log.Trace().Msg("OnEvent: PongEvent")
	})
	t := time.NewTicker(config.Get().GetKeepalive() * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Trace().Msg("OnEvent: PingEvent")
			cpc.socket.Emit(socketio.PingEvent)
		case <-ctx.Done():
			return
		}
	}
}

// Close closes the socket.io connection
func (cpc *ControlPlanClient) Close() {
	cpc.socket.Disconnect()
	cpc.socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
	cpc.manager.Close()
}

// getEioConfig return the socket.io underlying configuration
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
