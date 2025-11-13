package relay

import (
	"Goauld/agent/config"
	"Goauld/agent/control"
	"Goauld/agent/ssh/transport"
	"Goauld/common/log"
	commonnet "Goauld/common/net"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	socketio "Goauld/common/socket.io"

	gosio "github.com/hazegard/socket.io-go"
	eio "github.com/hazegard/socket.io-go/engine.io"
)

// ControlRouter represent the socket.io server.
type ControlRouter struct {
	Server       *gosio.Server
	Mode         string
	DnsTransport *transport.DNSSH
}

func (router *ControlRouter) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = commonnet.HTTP10ToHTTP11FakeUpgrader(r)
	id := r.PathValue("agentId")
	if !md5Re.MatchString(id) {
		http.NotFound(w, r)

		return
	}
	router.Server.ServeHTTP(w, r)
}

var md5Re = regexp.MustCompile(`\A(?i:[a-f0-9]{32})\z`)

// InitSocketIORelayServer initialize the relay server socket.io used to manage the agents.
func InitSocketIORelayServer(ctx context.Context, mode string, dnsTransport *transport.DNSSH) (*ControlRouter, error) {
	io := gosio.NewServer(&gosio.ServerConfig{
		EIO: eio.ServerConfig{
			Authenticator: func(_ http.ResponseWriter, _ *http.Request) bool {
				return true
			},
			WebTransportServer:     nil,
			WebSocketAcceptOptions: nil,
			DisableMaxBufferSize:   true,
		},
	})
	socketIO := &ControlRouter{
		Server:       io,
		Mode:         mode,
		DnsTransport: dnsTransport,
	}
	socketIO.Setup(ctx, io.Of("/"))
	err := io.Run()
	if err != nil {
		return nil, fmt.Errorf("error intializing socket.io: %w", err)
	}

	return socketIO, nil
}

// Setup configures the socket.io server.
func (sio *ControlRouter) Setup(ctx context.Context, root *gosio.Namespace) {
	root.OnConnection(func(agent2Relay gosio.ServerSocket) {
		agent2Relay.OnEvent(socketio.RegisterEvent.ID(), func(data any) {
			u := "http://" + strings.TrimPrefix(strings.TrimPrefix(config.Get().SocketIoURL("00000000000000000000000000000000"), "https://"), "http://")

			cfg, err := control.GetRelayEioConfig(sio.Mode, sio.DnsTransport)
			if err != nil {
				log.Error().Err(err).Msgf("Error getting relay config")
				agent2Relay.Disconnect(true)

				return
			}
			manager := gosio.NewManager(u, cfg)

			relay2Server := manager.Socket("/", nil)
			manager.OnError(func(err error) {
				log.Trace().Msg("OnError")
				log.Error().Err(err).Msgf("Error occurred  %s", u)
			})
			manager.OnReconnect(func(attempt uint32) {
				log.Trace().Msg("OnReconnect")
				log.Warn().Msgf("Reconnected to the control server %s, attempts N° %d", u, attempt)
			})
			SetupProxy(relay2Server, agent2Relay, u)
			relay2Server.Emit(socketio.RegisterEvent.ID(), data)
			go func() {
				<-ctx.Done()
				relay2Server.Disconnect()
				agent2Relay.Disconnect(true)
			}()
		})
	})
}

func SetupProxy(relay2Server gosio.ClientSocket, agent2Relay gosio.ServerSocket, u string) {
	relay2Server.OnConnect(func() {
		log.Trace().Msg("OnConnect")
		log.Info().Msgf("Connected to the control server %s", u)
	})
	relay2Server.OnConnectError(func(err any) {
		log.Trace().Msg("OnConnectError")
		log.Error().Msgf("Error occurred connecting to %s (%v)", u, err)
	})

	// RegisterEvent is emitted by the agent when connecting
	// The data sent is
	// - the ID of the agent (in cleartext);
	// - the name of the agent (encrypted using the age public key embedded in the agent);
	// - the shared encryption key (encrypted using the age public key embedded in the agent).
	// The shared encryption key will be used to encrypt the next events

	// SendAgentDataEvent is emitted by the agent to send the SSH password accepted by the agent
	// when connecting using the remote port forwarding
	agent2Relay.OnEvent(socketio.SendAgentDataEvent.ID(), func(data any) {
		log.Trace().Str("Event", "SendAgentDataEvent").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.SendAgentDataEvent.ID(), data)
	})

	// DeregisterEvent is sent by the agent when disconnecting
	agent2Relay.OnEvent(socketio.DeregisterEvent.ID(), func(_ socketio.Deregister) {
		log.Trace().Str("Event", "SendAgentDataEvent").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.DeregisterEvent.ID())
	})

	// PingEvent is sent at a regular interval by the agent to keep the connection active
	agent2Relay.OnEvent(socketio.PingEvent.ID(), func(_ socketio.Deregister) {
		log.Trace().Str("Event", "PingEvent").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.PingEvent.ID())
	})

	agent2Relay.OnEvent(socketio.ExitSuccess.ID(), func() {
		log.Trace().Str("Event", "ExitSuccess").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.ExitSuccess.ID())
	})

	agent2Relay.OnEvent(socketio.SendRemotePortForwardingDataEvent.ID(), func(data []byte) {
		log.Trace().Str("Event", "SendRemotePortForwardingDataEvent").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.SendRemotePortForwardingDataEvent.ID(), data)
	})

	agent2Relay.OnEvent(socketio.Disconnect.ID(), func(data []byte) {
		log.Trace().Str("Event", "Disconnect").Str("From", "Agent").Msg("Relay Event")
		relay2Server.Emit(socketio.Disconnect.ID(), data)
	})

	relay2Server.OnEvent(socketio.PongEvent.ID(), func(data any) {
		log.Trace().Str("Event", "PongEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.PongEvent.ID(), data)
	})

	relay2Server.OnEvent(socketio.SendSSHPrivateKeyEvent.ID(), func(data any) {
		log.Trace().Str("Event", "SendSSHPrivateKeyEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendSSHPrivateKeyEvent.ID(), data)
	})

	// SendSSHHPrivateKeyError Logs when the server returns an error
	relay2Server.OnEvent(socketio.SendSSHHPrivateKeyError.ID(), func() {
		log.Trace().Str("Event", "SendSSHHPrivateKeyError").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendSSHHPrivateKeyError.ID())
	})

	// VersionEvent sends the current server version
	// To display a message to the user if the server and the agent version mismatch
	relay2Server.OnEvent(socketio.VersionEvent.ID(), func(srvVersion any) {
		log.Trace().Str("Event", "VersionEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.VersionEvent.ID(), srvVersion)
	})

	// SendSSHPrivateKeySuccess Logs when the server returns no error
	relay2Server.OnEvent(socketio.SendSSHPrivateKeySuccess.ID(), func() {
		log.Trace().Str("Event", "SendSSHPrivateKeySuccess").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendSSHPrivateKeySuccess.ID())
	})

	// SendAgentDataError Logs when the server returns an error
	relay2Server.OnEvent(socketio.SendAgentDataError.ID(), func() {
		log.Trace().Str("Event", "SendAgentDataError").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendAgentDataError.ID())
	})

	// SendAgentDataSuccess Logs when the server returns no error
	// As it complete the configuration steps between the agent and the server
	relay2Server.OnEvent(socketio.SendAgentDataSuccess.ID(), func() {
		log.Trace().Str("Event", "SendAgentDataSuccess").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendAgentDataSuccess.ID())
	})

	// RegisterError fire when an error occurs on the server side when the agent registers
	relay2Server.OnEvent(socketio.RegisterError.ID(), func(data socketio.SioError) {
		log.Trace().Str("Event", "RegisterError").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.RegisterError.ID(), data)
	})

	// ExitEvent is sent by the server when the agent is requested to exit
	relay2Server.OnEvent(socketio.ExitEvent.ID(), func(doExit bool) {
		log.Trace().Str("Event", "ExitEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.ExitEvent.ID(), doExit)
	})

	// ExitEvent is sent by the server when the agent is requested to exit
	relay2Server.OnEvent(socketio.SendRemotePortForwardingDataSuccess.ID(), func(doExit bool) {
		log.Trace().Str("Event", "SendRemotePortForwardingDataSuccess").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendRemotePortForwardingDataSuccess.ID(), doExit)
	})

	// AlreadyConnectedEvent is sent by the server when the agent is already running.
	// The agent should exit
	relay2Server.OnEvent(socketio.AlreadyConnectedEvent.ID(), func() {
		log.Trace().Str("Event", "AlreadyConnectedEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.AlreadyConnectedEvent.ID())
	})

	relay2Server.OnEvent(socketio.PasswordValidationRequestEventRelay.ID(), func(data socketio.RelayEvent) {
		log.Trace().Str("Event", "PasswordValidationRequestEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.PasswordValidationRequestEvent.ID(), data.Data)

		agent2Relay.OnceEvent(data.ID, func(resp any) {
			log.Trace().Str("Event", data.ID).Str("From", "Agent").Msg("Relay Event")
			relay2Server.Emit(data.ID, resp)
		})
	})

	relay2Server.OnEvent(socketio.SendAgentDataEvent.ID(), func(data any) {
		log.Trace().Str("Event", "SendAgentDataEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.SendAgentDataEvent.ID(), data)
	})

	relay2Server.OnEvent(socketio.ClipboardContentEvent.ID(), func(data []byte) {
		log.Trace().Str("Event", "ClipboardContentEvent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.ClipboardContentEvent.ID(), data)
	})

	relay2Server.OnEvent(socketio.CopyClipboardRequestEventRelay.ID(), func(data socketio.RelayEvent) {
		log.Trace().Str("Event", "CopyClipboardRequestEventRelay").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.CopyClipboardRequestEvent.ID(), data.Data)

		agent2Relay.OnceEvent(data.ID, func(resp any) {
			log.Trace().Str("Event", "OnEvent").Str("From", "Agent").Msg("Relay Event")
			relay2Server.Emit(data.ID, resp)
		})
	})

	relay2Server.OnEvent(socketio.PingIsAliveRelay.ID(), func(data socketio.RelayEvent) {
		log.Trace().Str("Event", "PingIsAliveRelay").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.PingIsAlive.ID(), data)

		agent2Relay.OnceEvent(data.ID, func(resp any) {
			log.Trace().Str("Event", "OnEvent").Str("From", "Agent").Msg("Relay Event")
			relay2Server.Emit(data.ID, resp)
		})
	})

	relay2Server.OnEvent(socketio.WireguardPeer.ID(), func(data []byte) {
		log.Trace().Str("Event", "WireguardPeer").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.WireguardPeer.ID(), data)
	})

	relay2Server.OnEvent(socketio.ReceiveFatAgent.ID(), func(data []byte) {
		log.Trace().Str("Event", "ReceiveFatAgent").Str("From", "Server").Msg("Relay Event")
		agent2Relay.Emit(socketio.ReceiveFatAgent.ID(), data)
	})

	agent2Relay.OnDisconnect(func(reason gosio.Reason) {
		log.Trace().Str("Event", "Disconnect").Str("Reason", string(reason)).Str("From", "agent").Msg("Relay Event")
		relay2Server.Disconnect()
	})

	relay2Server.Connect()
}
