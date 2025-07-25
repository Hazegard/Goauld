package control

import (
	"Goauld/common"
	"Goauld/common/utils"
	"context"
	"errors"
	"fmt"
	"github.com/xtaci/smux"
	"strings"
	"time"

	"Goauld/agent/config"
	"Goauld/agent/proxy"
	"Goauld/common/crypto"
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"Goauld/common/ssh"
	sio "github.com/karagenc/socket.io-go"
	eio "github.com/karagenc/socket.io-go/engine.io"
	"github.com/quic-go/webtransport-go"
	"nhooyr.io/websocket"
)

// ControlPlanClient Handle the socket.io interaction regarding the management of the agent
type ControlPlanClient struct {
	manager      *sio.Manager
	socket       sio.ClientSocket
	configDone   chan<- struct{}
	ctx          context.Context
	url          string
	canceler     *utils.GlobalCanceler
	errorCounter int
}

// NewControlPlanClient returns a new ControlPlanClient
func NewControlPlanClient(ctx context.Context, configDone chan<- struct{}, canceler *utils.GlobalCanceler) *ControlPlanClient {
	return &ControlPlanClient{
		ctx:        ctx,
		url:        config.Get().SocketIoUrl(),
		configDone: configDone,
		canceler:   canceler,
	}
}

// InitStrategy is a struc holding the name of the transport as well
// as the function that will be used to initialize the socket.io connection
type InitStrategy struct {
	Name     string
	InitFunc CpcStarter
}

// CpcStarter is a function that will be used to initialize the socket.io connection
// It returns an error if the connection failed
type CpcStarter func(*ControlPlanClient, chan<- struct{}, chan<- error) error

// Init tries to connect to the control plan using the different strategies (CpcStarter)
// A successful connection will send a signal using the configDone channel
func Init(ctx context.Context, globalCanceler *utils.GlobalCanceler, configDone chan<- struct{}, controlErr chan<- error, CpcStarter CpcStarter) (*ControlPlanClient, error) {
	ctx, cancel := context.WithCancel(ctx)
	controlPlanClient := NewControlPlanClient(ctx, configDone, globalCanceler)
	chanErr := make(chan error)
	chanSuccess := make(chan struct{})
	err := CpcStarter(controlPlanClient, chanSuccess, chanErr)
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

// InitWs tries to connect to the control plan using the websocket transport
func (cpc *ControlPlanClient) InitWs(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"websocket"})
	return cpc.init(cfg, success, chanErr)
}

// InitWsUpgrade tries to connect to the control plan using the http to websocket upgrade transport
func (cpc *ControlPlanClient) InitWsUpgrade(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling", "websocket"})
	return cpc.init(cfg, success, chanErr)
}

// InitPolling tries to connect to the control plan using the HTTP long polling transport
func (cpc *ControlPlanClient) InitPolling(success chan<- struct{}, chanErr chan<- error) error {
	cfg := getEioConfig([]string{"polling"})
	return cpc.init(cfg, success, chanErr)
}

// InitOverDns tries to connect to the control plan using the DNS transport
func (cpc *ControlPlanClient) InitOverDns(session *smux.Stream, success chan<- struct{}, chanErr chan<- error) error {
	_, err := session.Write([]byte(config.Get().Id))
	// DNS MODE means we are using http to simplify the exchanges
	u := strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoUrl(), "https://"), "http://")
	cpc.url = fmt.Sprintf("http://%s", u)
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %v", err)
	}
	_, err = session.Write([]byte{'C'})
	if err != nil {
		return fmt.Errorf("error writing id to DNS tunnelled session: %v", err)
	}
	cfg := getDnsEioConfig(session)
	return cpc.init(cfg, success, chanErr)
}

// Init initializes the socket.io handlers
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
		log.Error().Msgf("Error occured connecting to %s (%v)", cpc.url, err)
		chanErr <- fmt.Errorf("error connecting to %s (%v)", cpc.url, err)
	})

	manager.OnError(func(err error) {
		log.Trace().Msg("OnError")
		log.Error().Err(err).Msgf("Error occured  %s", cpc.url)
		cpc.ErrorPlusPlus()
	})
	manager.OnReconnect(func(attempt uint32) {
		cpc.canceler.Restart("Control socket disconnected")
		log.Trace().Msg("OnReconnect")
		log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", cpc.url, attempt)
	})

	// SendSshPrivateKeyEvent is sent by the server after the client sends the RegisterEvent event
	// this event contains the encrypted SSH private key used by the agent to authenticate on the
	// SSHD server.
	// Once received, the agent sends its SSHD password to the server using the SendAgentDataEvent event
	socket.OnEvent(socketio.SendSshPrivateKeyEvent, func(data []byte) {
		log.Trace().Msg("OnEvent: SendSshPrivateKeyEvent")
		log.Trace().Msgf("SshPrivateKeyEvent: data received")
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

	// VersionEvent sends the current server version
	// To display a message to the user if the server and the agent version mismatch
	socket.OnEvent(socketio.VersionEvent, func(srvVersion common.JVersion) {
		agentVersion := common.JsonVersion()
		if agentVersion.Compare(srvVersion) != 0 {
			log.Warn().Err(fmt.Errorf("mismatch version")).Str("Server", srvVersion.Version).Str("Agent", agentVersion.Version).Msgf("Version mismatch")
			log.Trace().Str("ServerCommit", srvVersion.Commit).Str("AgentCommit", agentVersion.Commit).Msgf("Version mismatch")
			log.Trace().Str("ServerDate", srvVersion.Date).Str("AgentDate", agentVersion.Date).Msgf("Version mismatchs")
		}
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
	// As it complete the configuration steps between the agent and the server
	socket.OnEvent(socketio.SendAgentDataSuccess, func() {
		log.Trace().Msg("OnEvent: SendAgentDataSuccess")
		cpc.configDone <- struct{}{}
		log.Trace().Msg("OnEvent: SendAgentDataSuccess done")
	})

	// RegisterError fire when an error occurs on the server side when the agent registers
	socket.OnEvent(socketio.RegisterError, func(data socketio.SioError) {
		if strings.Contains(data.Message, "UNIQUE constraint failed: agents.name") {
			log.Error().Err(errors.New("Agent Name already used, either delete the corresponding agent in the TUI or rename this agent")).Msgf("RegisterError")
			cpc.canceler.Exit("Agent Name already used")
			cpc.Close()
		} else {
			log.Error().Err(errors.New(data.Message)).Msgf("Error occured %s", "RegisterError")
			log.Info().Msgf("Restarting...")
			socket.Disconnect()

			cpc.canceler.Restart("Error occurs on the server while registering")
			cpc.Close()
		}
	})

	// ExitEvent is sent by the server when the agent is requested to exit
	socket.OnEvent(socketio.ExitEvent, func(doExit bool) {
		log.Info().Msg("OnEvent: Exit requested")
		socket.Emit(socketio.ExitSuccess)
		socket.Disconnect()
		if doExit {
			cpc.canceler.Exit("Server requested exit")
		}
		cpc.canceler.Restart("Server requested restart")
		cpc.Close()
	})

	// AlreadyConnectedEvent is sent by the server when the agent is already running.
	// The agent should exit
	socket.OnEvent(socketio.AlreadyConnectedEvent, func() {
		log.Info().Msg("AlreadyConnectedEvent: Exit requested because agent is already running")
		socket.Emit(socketio.ExitSuccess)
		socket.Disconnect()
		cpc.canceler.Exit("The agent is already connected")
	})

	socket.OnEvent(socketio.PasswordValidationRequestEvent, func(data []byte) {

		log.Trace().Msg("OnEvent: PasswordValidationRequestEvent")
		passwordValidationReq, err := socketio.DecryptPasswordValidationRequest(data, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: DecryptPasswordValidationRequest")
			return
		}
		res := passwordValidationReq.Password == config.Get().PrivateSshdPassword()

		response, err := socketio.NewEncryptPasswordValidationResponse(res, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: EncryptPasswordValidationRequest")
		}
		socket.Emit(passwordValidationReq.EventId, response)
		log.Trace().Msgf("Emit: %s", passwordValidationReq.EventId)
	})

	cpc.socket = socket
	cpc.manager = manager
	return nil
}

// Start starts the socket and initiates the configuration exchange with the server
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
	// starts the keepalive in the background
	go cpc.keepAliveLoop(cpc.ctx)
	log.Debug().Msgf("Connected to the control server %s", cpc.url)
	log.Trace().Msg("Event send: RegisterEvent")
	// Waits for an error or the end of the socket
	<-cpc.ctx.Done()
	log.Warn().Msgf("Shutting done the socketio control socket")
	cpc.socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
	log.Trace().Msg("Event send: Disconnect")
	cpc.socket.Disconnect()
	return nil
}

// SendPorts sends the remote ports used by the agent
func (cpc *ControlPlanClient) SendPorts(rpf []ssh.RemotePortForwarding) error {
	data, err := socketio.EncryptRemotePortForwardingMessage(rpf, config.Get().Cryptor)
	if err != nil {
		return fmt.Errorf("error encrypting remote port forwarding message: %v", err)
	}

	success := make(chan struct{}, 1)
	// SendRemotePortForwardingDataError is sent by the server when the forwarding ports
	// are successfully received by the server
	cpc.socket.OnEvent(socketio.SendRemotePortForwardingDataSuccess, func() {
		log.Info().Msgf("SendRemotePortForwardingDataSuccess successfully sent")
		success <- struct{}{}
	})
	defer cpc.socket.OffEvent(socketio.SendRemotePortForwardingDataSuccess)
	cpc.socket.Emit(socketio.SendRemotePortForwardingDataEvent, data)
	<-success
	return nil
}

// KeepAliveLoop starts a keepalive loop that will periodically send ping
//
// to keep alive the connection
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

// ErrorPlusPlus handle when an error occurs on the socket.io side
// If more than 5 errors occur, the agent will automatically restart
// See to check the errors in a given time, reset the counter after some time
func (cpc *ControlPlanClient) ErrorPlusPlus() {
	cpc.errorCounter++
	if cpc.errorCounter > 5 {
		log.Warn().Msgf("Error occured %d times, restarting...", cpc.errorCounter)
		cpc.canceler.Restart(fmt.Sprintf("Control sockets crashed %d times", cpc.errorCounter))
		cpc.Close()
	}
}

// Close closes the socket.io connection
func (cpc *ControlPlanClient) Close() {
	cpc.socket.Disconnect()
	cpc.socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
	cpc.manager.Close()
}

// getEioConfig return the socket.io underlying configuration
func getEioConfig(transport []string) *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Msg("Client transport upgrade done")
			},
			HTTPTransport: proxy.NewTransportProxy(),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: proxy.NewHttpClientProxy(nil),
				HTTPHeader: proxy.NewHeaderMap(),
			},
			WebTransportDialer: &webtransport.Dialer{
				TLSClientConfig: proxy.NewTlsConfig(),
			},
			Transports: transport, // []string{"polling"}, //, "websocket", "webtransport"},
			// Debugger:   sio.NewPrintDebugger(),
		},
	}
}

// getEioConfig return the socket.io underlying configuration
func getDnsEioConfig(session *smux.Stream) *sio.ManagerConfig {

	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				log.Trace().Msg("Client transport upgrade done")
			},
			HTTPTransport: NewSmuxTransport(session),
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: newSmuxHTTPandHTTPSClient(session),
			},
			// When tunneling over DNS, if we use polling only or polling then websocket upgrade,
			// The tunnel fails to establish properly as the server responds to unwanted content to the open HTTP socket.
			// Here we use the full duplex websocket mechanism to ensure that the tunnel is properly working
			// On the client side
			Transports: []string{"websocket"}, //, "websocket"},
			// Debugger:   sio.NewPrintDebugger(),
		},
	}
}
