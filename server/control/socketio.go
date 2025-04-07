package control

import (
	"fmt"
	"net/http"
	"time"

	common_net "Goauld/common/net"

	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"

	gosio "github.com/karagenc/socket.io-go"
)

type SocketIO struct {
	db         *persistence.DB
	agentStore *store.AgentStore
	Server     *gosio.Server
}

// ServeHTTP serves the socket.IO HTTP server
func (sio *SocketIO) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = common_net.Http10ToHttp11FakeUpgrader(r)
	sio.Server.ServeHTTP(w, r)
}

// InitSocketIOServer initialize the server socket.io used to manage the agents
func InitSocketIOServer(agentStore *store.AgentStore, db *persistence.DB) (*SocketIO, error) {
	io := gosio.NewServer(&gosio.ServerConfig{})
	socketIO := &SocketIO{
		agentStore: agentStore,
		db:         db,
		Server:     io,
	}
	socketIO.Setup(io.Of("/"))
	err := io.Run()
	if err != nil {
		return nil, fmt.Errorf("error intializing socket.io: %s", err)
	}

	return socketIO, nil
}

func (sio *SocketIO) Setup(root *gosio.Namespace) {
	root.OnConnection(func(socket gosio.ServerSocket) {
		// RegisterEvent is emitted by the agent when connecting
		// The data sent is
		// - the ID of the agent (in cleartext)
		// - the name of the agent (encrypted using the age public key embedded in the agent)
		// - the shared encryption key (encrypted using the age public key embedded in the agent)
		// The shared encryption key will be used to encryt the next events
		socket.OnEvent(socketio.RegisterEvent, func(data socketio.Register) {
			// Retrieving the agent id from the received data
			log.Debug().Str("Agent.Id", data.Id).Msg("socketio.RegisterEvent")

			// Decrypting the agent name using the age private key
			log.Trace().Str("Agent.name", "?????").Str("Agent.Id", data.Id).Msg("START socketio.decrypting agent name")
			agentName, err := config.Get().Decrypt(data.Name)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting name: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				log.Error().Str("Agent.name", "?????").Str("Agent.Id", data.Id).Err(err).Msg("socketio.RegisterError error decrypting agent name")
				return
			}

			// Retrieve the agent from the database
			// If no agent corresponding to this ID exists
			// an empty one that will be populated later is returned
			agent, err := sio.db.FindOrCreate(data.Id, agentName)
			if err != nil {
				errorMsg := fmt.Errorf("error retrieving agent: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterError,
				})
				log.Error().Str("Agent.name", "").Str("Agent.Id", data.Id).Err(err).Msg("socketio.RegisterError retrieving agent")
				return
			}
			if agent.Connected || agent.SocketId != "" {
				log.Error().Str("Agent.name", "").Str("Agent.Id", data.Id).Err(err).Msg("agent already connected... emitting kill")
				socket.Emit(socketio.AlreadyConnectedEvent)
				return
			}

			// Decrypting the shared key using the age private key
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent decrypting shared key")
			sharedSecret, err := config.Get().Decrypt(data.SharedKey)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting shared secret (%s): %s", data.Id, err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError decrypting shared secret")
				return
			}

			// Saving the agent information in the database
			agent.SetSharedSecret(sharedSecret)
			agent.SetName(agentName)
			agent.SetConnect()
			agent.SocketId = string(socket.ID())
			err = sio.db.UpdateAgentField(agent, "SharedSecret", "Name", "Connected", "SocketId")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating agent")
			}

			// Adding the agent to the in memory storage that keeps information related to
			// the server components
			sio.agentStore.SioAddAgent(agent, socket)

			// If no private key exists (ie a new agent is connecting)
			// generate a private key
			if agent.PrivateKey == "" {
				log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent no ssh keys, generating...")
				err := agent.InitKeys()
				if err != nil {
					socket.Emit(socketio.RegisterError, socketio.SioError{
						Message: "error generating keys",
						Code:    socketio.RegisterError,
					})
					log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error generating ssh keys")
					return
				}
				err = sio.db.UpdateAgentField(agent, "PrivateKey", "PublicKey")
				if err != nil {
					log.Error().Err(err).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating ssh keys")
				}
			}

			// Get the encryption library to encrypt the private key
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent initiating cryptor")
			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting fields",
					Code:    socketio.RegisterError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error encrypting ssh keys")
				return
			}

			// Encrypting the SSH private key using the shared key previously send by the agent
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent encrypting ssh private key")
			message, err := socketio.NewEncryptedSshPrivateKeyMessage(agent.PrivateKey, cryptor)
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting keys",
					Code:    socketio.RegisterError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error encrypting ssh keys")
				return
			}

			// Sending the encrypted private key to the agent
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent sending encrypted ssh private key")
			socket.Emit(socketio.SendSshPrivateKeyEvent, message)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.SendSshPrivateKeyEvent")
		})

		// SendAgentDataEvent is emitted by the agent to send the SSH password accepted by the agent
		// when connecting using the remote port forwarding
		socket.OnEvent(socketio.SendAgentDataEvent, func(data []byte) {
			// Retrieving the agent from the database
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.SendAgentDataEvent")

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendAgentDataError, socketio.SioError{
					Message: "error geting decryptor for agent ssh password",
					Code:    socketio.SendAgentDataError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error decrypting ssh password")
				return
			}
			// Decrypting the SSH password using the shared encryption key
			agentData, err := socketio.DecryptAgentSshPasswordMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendAgentDataError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendAgentDataError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error decrypting ssh password")
				return
			}
			// Saving the SSH password in the database
			agent.SetSshPassword(agentData.AgentSshPassword)
			agent.Platform = agentData.Platform
			agent.Architecture = agentData.Architecture
			agent.Username = agentData.Username
			agent.Hostname = agentData.Hostname
			agent.Path = agentData.Path
			agent.IPs = agentData.IPs

			err = sio.db.UpdateAgentField(agent, "SshPasswd", "Platform", "Architecture", "Username", "Hostname", "Path", "IPs")
			if err != nil {
				log.Error().Err(err).Str("Agent.Id", agent.Id).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating agent")
			}
			socket.Emit(socketio.SendAgentDataSuccess)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("END socketio.SendAgentDataEvent")
		})

		// DeregisterEvent is sent by the agent when disconnecting
		socket.OnEvent(socketio.DeregisterEvent, func(data socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			agent.SetDisconnect()
			err := sio.db.UpdateAgentField(agent, "Connected")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.RegisterError updating agent")
			}
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.DeregisterEvent")
		})

		// PingEvent is sent at regular interval by the agent to keep the connection active
		socket.OnEvent(socketio.PingEvent, func(data socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("Received socketio.PingEvent")
			socket.Emit(socketio.PongEvent)
			agent.LastUpdated = time.Now()
			_ = sio.db.UpdateAgentField(agent, "LastUpdated")
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("Sent socketio.PongEvent")
		})

		socket.OnEvent(socketio.ExitSuccess, func() {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Info().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("Agent exited")
		})

		socket.OnEvent(socketio.SendRemotePortForwardingDataEvent, func(data []byte) {
			// Retrieving the agent from the database
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.SendAgentDataEvent")

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError, socketio.SioError{
					Message: "error geting decryptor for agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.SendRemotePortForwardingDataEvent error decrypting ssh password")
				return
			}

			// Decrypting the SSH password using the shared encryption key
			rpfData, err := socketio.DecryptRemotePortForwardingMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.SendRemotePortForwardingDataEvent error decrypting ssh password")
				return
			}
			agent.SetRemotePortForwarding(rpfData)
			err = sio.db.UpdateAgentField(agent, "RemotePortForwarding")
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError,
				})
				log.Error().Err(err).Str("Agent.Id", agent.Id).Str("Agent.Name", agent.Name).Msg("socketio.SendRemotePortForwardingDataEvent error updating agent")
			}
			socket.Emit(socketio.SendRemotePortForwardingDataSuccess)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("END socketio.SendRemotePortForwardingDataEvent")
		})

		socket.OnDisconnect(func(reason gosio.Reason) {
			agent := sio.agentStore.SioGetAgent(socket)
			if agent.SocketId != string(socket.ID()) {
				log.Info().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msgf("socketio.Disconnect: %s", "already connected")
				return
			}
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msgf("socketio.Disconnect: %s", reason)

			// Close all active connections related to the agent
			err := sio.agentStore.CloseAgentConnections(agent.Id)
			if err != nil {
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.Disconnect: error closing agent")
			}
			// Remove the agent from the im memory store
			sio.agentStore.SioRemoveAgent(socket)

			err = sio.db.SetAgentSshMode(agent.Id, "OFF")
			if err != nil {
				log.Warn().Err(err).Str("Agent.Name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.Disconnect: error setting agent connection mode")
			}
			agent.SetDisconnect()
			agent.SocketId = ""
			err = sio.db.UpdateAgentField(agent, "Connected", "SocketId")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.Disconnect updating agent")
			}
		})
	})
}
