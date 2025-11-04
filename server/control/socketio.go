// Package control holds the socket.io server
package control

import (
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/wireguard"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"time"

	eio "github.com/hazegard/socket.io-go/engine.io"
	"github.com/google/uuid"

	commonnet "Goauld/common/net"

	"Goauld/common/log"
	"Goauld/server/config"
	"Goauld/server/persistence"
	"Goauld/server/store"

	socketio "Goauld/common/socket.io"

	gosio "github.com/hazegard/socket.io-go"
)

// SocketIO represent the socket.io server.
type SocketIO struct {
	db         *persistence.DB
	agentStore *store.AgentStore
	Server     *gosio.Server
}

// deprecated
// ServeHTTP serves the socket.IO HTTP server
// func (sio *SocketIO) ServeHTTP(w http.ResponseWriter, r *http.Request) {
// 	r = commonnet.HTTP10ToHTTP11FakeUpgrader(r)
// 	sio.Server.ServeHTTP(w, r)
// }

var md5Re = regexp.MustCompile(`\A(?i:[a-f0-9]{32})\z`)

// ServeHTTP serves the socket.IO HTTP server.
func (sio *SocketIO) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r = commonnet.HTTP10ToHTTP11FakeUpgrader(r)
	id := r.PathValue("agentId")
	if !md5Re.MatchString(id) {
		http.NotFound(w, r)

		return
	}
	sio.agentStore.AddRemote(id, r.RemoteAddr)

	sio.Server.ServeHTTP(w, r)
}

// InitSocketIOServer initialize the server socket.io used to manage the agents.
func InitSocketIOServer(agentStore *store.AgentStore, db *persistence.DB) (*SocketIO, error) {
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
	socketIO := &SocketIO{
		agentStore: agentStore,
		db:         db,
		Server:     io,
	}
	socketIO.Setup(io.Of("/"))
	err := io.Run()
	if err != nil {
		return nil, fmt.Errorf("error intializing socket.io: %w", err)
	}

	return socketIO, nil
}

// Setup configures the socket.io server.
func (sio *SocketIO) Setup(root *gosio.Namespace) {
	root.OnConnection(func(socket gosio.ServerSocket) {
		// RegisterEvent is emitted by the agent when connecting
		// The data sent is
		// - the ID of the agent (in cleartext);
		// - the name of the agent (encrypted using the age public key embedded in the agent);
		// - the shared encryption key (encrypted using the age public key embedded in the agent).
		// The shared encryption key will be used to encrypt the next events
		socket.OnEvent(socketio.RegisterEvent.ID(), func(data socketio.Register) {
			// Retrieving the agent id from the received data
			log.Debug().Str("Agent.ID", data.ID).Msg("socketio.RegisterEvent")

			socket.Emit(socketio.VersionEvent.ID(), common.JSONVersion())

			// Decrypting the agent name using the age private key
			log.Trace().Str("Agent.name", "?????").Str("Agent.ID", data.ID).Msg("START socketio.decrypting agent name")
			agentName, err := config.Get().Decrypt(data.Name)

			if err != nil {
				errorMsg := fmt.Errorf("error decrypting name: %w", err)
				socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent.ID(),
				})
				log.Error().Str("Agent.name", "?????").Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError error decrypting agent name")

				return
			}
			if data.Load {
				// The agent is a loader
				sio.HandleLoader(socket, data, agentName)

				return
			}

			// Retrieve the agent from the database
			// If no agent corresponding to this ID exists,
			// an empty one that will be populated later is returned
			agent, err := sio.db.FindOrCreate(data.ID, agentName)
			if err != nil {
				errorMsg := fmt.Errorf("error retrieving agent: %w", err)
				socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterError.ID(),
				})
				log.Error().Str("Agent.name", "").Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError retrieving agent")

				return
			}
			if agent.Connected || agent.SocketID != "" {
				log.Error().Str("Agent.name", "").Str("Agent.ID", data.ID).Err(err).Msg("agent already connected... emitting kill")
				socket.Emit(socketio.AlreadyConnectedEvent.ID())

				return
			}

			// Decrypting the shared key using the age private key
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("START socketio.RegisterEvent decrypting shared key")
			sharedSecret, err := config.Get().Decrypt(data.SharedKey)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting shared secret (%s): %w", data.ID, err)
				socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError decrypting shared secret")

				return
			}
			remote := sio.agentStore.GetRemote(data.ID)
			agent.RemoteAddr = remote

			// Saving the agent information in the database
			agent.SetSharedSecret(sharedSecret)
			agent.SetName(agentName)
			agent.SetConnect()
			agent.SocketID = string(socket.ID())
			err = sio.db.UpdateAgentField(agent, "SharedSecret", "Name", "Connected", "SocketID", "RemoteAddr")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating agent")
			}

			// Adding the agent to the in memory storage that keeps information related to
			// the server components
			sio.agentStore.SioAddAgent(agent, socket)

			// If no private key exists (i.e., a new agent is connecting)
			// generate a private key
			if agent.PrivateKey == "" {
				log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("START socketio.RegisterEvent no ssh keys, generating...")
				err := agent.InitKeys()
				if err != nil {
					socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
						Message: "error generating keys",
						Code:    socketio.RegisterError.ID(),
					})
					log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError error generating ssh keys")

					return
				}
				err = sio.db.UpdateAgentField(agent, "PrivateKey", "PublicKey")
				if err != nil {
					log.Error().Err(err).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating ssh keys")
				}
			}

			// Get the encryption library to encrypt the private key
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("START socketio.RegisterEvent initiating cryptor")
			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
					Message: "error encrypting fields",
					Code:    socketio.RegisterError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError error encrypting ssh keys")

				return
			}

			// Encrypting the SSH private key using the shared key previously send by the agent
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("START socketio.RegisterEvent encrypting ssh private key")
			message, err := socketio.NewEncryptedSSHPrivateKeyMessage(agent.PrivateKey, cryptor)
			if err != nil {
				socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
					Message: "error encrypting keys",
					Code:    socketio.RegisterError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError error encrypting ssh keys")

				return
			}

			// Sending the encrypted private key to the agent
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("START socketio.RegisterEvent sending encrypted ssh private key")
			socket.Emit(socketio.SendSSHPrivateKeyEvent.ID(), message)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.SendSSHPrivateKeyEvent")
		})

		// SendAgentDataEvent is emitted by the agent to send the SSH password accepted by the agent
		// when connecting using the remote port forwarding
		socket.OnEvent(socketio.SendAgentDataEvent.ID(), func(data []byte) {
			// Retrieving the agent from the database
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.SendAgentDataEvent")

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendAgentDataError.ID(), socketio.SioError{
					Message: "error getting decryptor for agent ssh password",
					Code:    socketio.SendAgentDataError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError error decrypting ssh password")

				return
			}
			// Decrypting the SSH password using the shared encryption key
			agentData, err := socketio.DecryptAgentSSHPasswordMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendAgentDataError.ID(), socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendAgentDataError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.RegisterError error decrypting ssh password")

				return
			}
			// Saving the SSH password in the database
			agent.SetSSHPassword(agentData.AgentSSHPassword)
			agent.Platform = agentData.Platform
			agent.Architecture = agentData.Architecture
			agent.Username = agentData.Username
			agent.Hostname = agentData.Hostname
			agent.Path = agentData.Path
			agent.IPs = agentData.IPs
			agent.HasStaticPassword = agentData.HasStaticPwd
			agent.Version = agentData.AgentVersion
			agent.WireguardPublicKey = agentData.WireguardPubKey
			agent.WireguardIP = agentData.WireguardIP

			err = sio.db.UpdateAgentField(agent, "SSHPasswd", "Platform", "Architecture", "Username", "Hostname", "Path", "IPs", "HasStaticPassword", "Version", "WireguardPublicKey", "WireguardIP")
			if err != nil {
				log.Error().Err(err).Str("Agent.ID", agent.ID).Str("Agent.Name", agent.Name).Msg("socketio.RegisterError updating agent")
			}
			socket.Emit(socketio.SendAgentDataSuccess.ID())
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("END socketio.SendAgentDataEvent")
		})

		// DeregisterEvent is sent by the agent when disconnecting
		socket.OnEvent(socketio.DeregisterEvent.ID(), func(_ socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			agent.SetDisconnect()
			err := sio.db.UpdateAgentField(agent, "Connected")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.RegisterError updating agent")
			}
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.DeregisterEvent")
		})

		// PingEvent is sent at a regular interval by the agent to keep the connection active
		socket.OnEvent(socketio.PingEvent.ID(), func(_ socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("Received socketio.PingEvent")
			socket.Emit(socketio.PongEvent.ID())
			agent.LastPing = time.Now()
			// We update the lastUpdated field in the database to show clients
			// that the connection with the agent is still working
			_ = sio.db.UpdateAgentFieldShadow(agent, "LastPing")
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("Sent socketio.PongEvent")
		})

		socket.OnEvent(socketio.ExitSuccess.ID(), func() {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Info().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("Agent exited")
		})

		socket.OnEvent(socketio.SendRemotePortForwardingDataEvent.ID(), func(data []byte) {
			// Retrieving the agent from the database
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.SendAgentDataEvent")

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError.ID(), socketio.SioError{
					Message: "error getting decryptor for agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.SendRemotePortForwardingDataEvent error decrypting ssh password")

				return
			}

			// Decrypting the SSH password using the shared encryption key
			rpfData, err := socketio.DecryptRemotePortForwardingMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError.ID(), socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError.ID(),
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.SendRemotePortForwardingDataEvent error decrypting ssh password")

				return
			}
			agent.SetRemotePortForwarding(rpfData)
			err = sio.db.UpdateAgentField(agent, "RemotePortForwarding")
			if err != nil {
				socket.Emit(socketio.SendRemotePortForwardingDataError.ID(), socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendRemotePortForwardingDataError.ID(),
				})
				log.Error().Err(err).Str("Agent.ID", agent.ID).Str("Agent.Name", agent.Name).Msg("socketio.SendRemotePortForwardingDataEvent error updating agent")
			}
			socket.Emit(socketio.SendRemotePortForwardingDataSuccess.ID())
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msg("END socketio.SendRemotePortForwardingDataEvent")
		})

		socket.OnDisconnect(func(reason gosio.Reason) {
			agent := sio.agentStore.SioGetAgent(socket)
			if agent.SocketID != string(socket.ID()) {
				log.Info().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msgf("socketio.Disconnect: %s", "already connected")

				return
			}
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Msgf("socketio.Disconnect: %s", reason)

			// Close all active connections related to the agent
			err := sio.agentStore.CloseAgentConnections(agent.ID)
			if err != nil {
				log.Error().Str("Agent.name", agent.Name).Str("Agent.ID", agent.ID).Err(err).Msg("socketio.Disconnect: error closing agent")
			}
			// Remove the agent from the im memory store
			sio.agentStore.SioRemoveAgent(socket)

			err = sio.db.SetAgentSSHMode(agent.ID, "OFF", "")
			if err != nil {
				log.Warn().Err(err).Str("Agent.Name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.Disconnect: error setting agent connection mode")
			}
			agent.SetDisconnect()
			agent.SocketID = ""
			err = sio.db.UpdateAgentField(agent, "Connected", "SocketID")
			if err != nil {
				log.Error().Err(err).Str("Agent.Name", agent.Name).Str("Agent.ID", agent.ID).Msg("socketio.Disconnect updating agent")
			}
		})
	})
}

// ValidateStaticPassword validate the provided agent matches the agent password
// It creates a custom random event tha will be listened on and send a PasswordValidationRequestResponse event to the agent.
// The agent will respond on the random event ID.
func ValidateStaticPassword(agent *persistence.Agent, socket gosio.Socket, hashAgentPwd string) bool {
	cryptor, err := agent.GetCryptor()
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msgf("Error getting crypto for agent (%s)", agent.Name)

		return false
	}
	if socket == nil {
		return false
	}

	id := uuid.NewString()
	eventID := fmt.Sprintf("%s@%s", socketio.PasswordValidationRequestResponse.ID(), id)
	chanResponse := make(chan bool, 1)
	socket.OnEvent(eventID, func(data []byte) {
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Msg("Event received")
		response, err := socketio.DecryptPasswordValidationResponse(data, cryptor)
		if err != nil {
			log.Error().Err(err).Msg("Error decrypting password validation response")
			chanResponse <- false

			return
		}
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Bool("Response", response.Response).Msg("Agent response")
		chanResponse <- response.Response
	})
	defer socket.OffEvent(eventID)

	encryptedPasswordValidationRequest, err := socketio.NewEncryptPasswordValidationRequest(hashAgentPwd, eventID, cryptor)
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Str("Event", eventID).Err(err).Msgf("Error encrypting password validation request")

		return false
	}

	socket.Emit(socketio.PasswordValidationRequestEvent.ID(), encryptedPasswordValidationRequest)
	var response bool
	select {
	case r := <-chanResponse:
		response = r
	case <-time.After(10 * time.Second):
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Msg("Timeout waiting for response")
		response = false
	}

	return response
}

func SetClipboard(agent *persistence.Agent, socket gosio.Socket, clip socketio.ClipboardMessage) bool {
	cryptor, err := agent.GetCryptor()
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msgf("Error getting crypto for agent (%s)", agent.Name)

		return false
	}
	if socket == nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msg("socket is nil")

		return false
	}
	data, err := socketio.NewEncryptedClipboardMessageEventMessage(clip, cryptor)
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msg("Error creating clipboard message event")

		return false
	}

	socket.Emit(socketio.ClipboardContentEvent.ID(), data)

	return true
}

func GetClipboard(agent *persistence.Agent, socket gosio.Socket, hashAgentPwd string) socketio.ClipboardMessage {
	cryptor, err := agent.GetCryptor()
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msgf("Error getting crypto for agent (%s)", agent.Name)

		return socketio.ClipboardMessage{
			Error:    true,
			ErrorMsg: fmt.Errorf("error getting crypto for agent (%s)", agent.Name),
		}
	}
	if socket == nil {
		return socketio.ClipboardMessage{
			Error:    true,
			ErrorMsg: errors.New("agent socket is nil"),
		}
	}

	id := uuid.NewString()
	eventID := fmt.Sprintf("%s@%s", socketio.ClipboardContentEvent.ID(), id)
	chanResponse := make(chan socketio.ClipboardMessage, 1)
	socket.OnEvent(eventID, func(data []byte) {
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Msg("Event received")
		response, err := socketio.DecryptClipboardMessageEventMessage(data, cryptor)
		if err != nil {
			log.Error().Err(err).Msg("Error decrypting clipboard message response")
			chanResponse <- socketio.ClipboardMessage{
				Error:    true,
				ErrorMsg: errors.New("error decrypting clipboard message response"),
			}

			return
		}
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Bool("Response", response.Error).Msg("Agent response")
		chanResponse <- response
	})
	defer socket.OffEvent(eventID)

	request := socketio.ClipboardRequestMessage{
		EventID:      eventID,
		HashPassword: hashAgentPwd,
	}

	encryptedClipboardRequest, err := socketio.NewEncryptedClipboardRequestMessageEventMessage(request, cryptor)
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Str("Event", eventID).Err(err).Msgf("Error encrypting password validation request")

		return socketio.ClipboardMessage{
			Error:    true,
			ErrorMsg: errors.New("error encrypting password validation request"),
		}
	}

	socket.Emit(socketio.CopyClipboardRequestEvent.ID(), encryptedClipboardRequest)
	var response socketio.ClipboardMessage
	select {
	case r := <-chanResponse:
		response = r
	case <-time.After(10 * time.Second):
		log.Debug().Str("Event", eventID).Str("Agent", agent.Name).Msg("Timeout waiting for response")
		response = socketio.ClipboardMessage{
			Error:    true,
			ErrorMsg: errors.New("timeout waiting for clipboard message response"),
		}
	}

	return response
}

func AddWGPeer(agent *persistence.Agent, socket gosio.Socket, wgPeer wireguard.WGConfig) bool {
	log.Info().Str("Agent.Name", agent.Name).Str("Wg IP", wgPeer.IP).Msg("Adding WG Peer")
	cryptor, err := agent.GetCryptor()
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msgf("Error getting crypto for agent (%s)", agent.Name)

		return false
	}
	if socket == nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msg("socket is nil")

		return false
	}
	data, err := socketio.NewEncryptedWGConfigEventMessage(wgPeer, cryptor)
	if err != nil {
		log.Debug().Str("Agent.Name", agent.Name).Err(err).Msg("Error creating clipboard message event")

		return false
	}

	socket.Emit(socketio.WireguardPeer.ID(), data)

	return true
}

func (sio *SocketIO) HandleLoader(socket gosio.ServerSocket, data socketio.Register, agentName string) {
	sharedSecret, err := config.Get().Decrypt(data.SharedKey)
	if err != nil {
		errorMsg := fmt.Errorf("error decrypting shared secret: %w", err)
		socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
			Message: errorMsg.Error(),
			Code:    socketio.RegisterEvent.ID(),
		})
		log.Error().Str("Type", "Loader").Str("Agent.name", "").Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError retrieving agent")
	}
	cryptor, err := crypto.NewCryptor(sharedSecret)
	if err != nil {
		errorMsg := fmt.Errorf("error decrypting shared secret: %w", err)
		socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
			Message: errorMsg.Error(),
			Code:    socketio.RegisterEvent.ID(),
		})
		log.Error().Str("Type", "Loader").Str("Agent.name", agentName).Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError decrypting shared secret")

		return
	}
	agentData, err := socketio.DecryptAgentSSHPasswordMessage(data.AgentData, cryptor)
	if err != nil {
		errorMsg := fmt.Errorf("error decrypting agent data: %w", err)
		socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
			Message: errorMsg.Error(),
			Code:    socketio.RegisterEvent.ID(),
		})
		log.Error().Str("Type", "Loader").Str("Agent.name", agentName).Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError decrypting agent data")
	}

	agentDB, err := sio.db.FindOrCreate(data.ID, agentName)
	if err != nil {
		log.Error().Str("Agent.name", "").Str("Agent.ID", data.ID).Err(err).Msg("socketio.RegisterError retrieving agent")

		return
	}
	agentDB.Version = agentData.AgentVersion
	agentDB.Version.Version = "jaffa/" + agentDB.Version.Version

	agentDB.Name = agentName
	agentDB.Platform = agentData.Platform
	agentDB.Architecture = agentData.Architecture
	agentDB.Path = agentData.Path
	agentDB.Hostname = agentData.Hostname
	agentDB.Username = agentData.Username

	agentDB.RemoteAddr = sio.agentStore.GetRemote(data.ID)

	sio.agentStore.SioAddAgent(agentDB, socket)

	err = sio.db.UpdateAgentField(agentDB, "Version", "Platform", "Architecture", "Path", "Hostname", "Username", "RemoteAddr")
	if err != nil {
		log.Error().Err(err).Str("Agent.Name", agentName).Msg("socketio.RegisterError updating agent")
	}
	binary := fmt.Sprintf("goauld_%s-%s", agentData.Platform, agentData.Architecture)
	if agentData.Platform == "windows" {
		binary += ".exe"
	}
	binaryData, err := os.ReadFile(filepath.Join(config.Get().BinariesPathLocation, binary))
	if err != nil {
		errorMsg := fmt.Errorf("error decrypting agent data: %w", err)
		socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
			Message: errorMsg.Error(),
			Code:    socketio.RegisterEvent.ID(),
		})
		log.Error().Str("Type", "Loader").Str("Agent.name", agentName).Str("Agent.ID", data.ID).Err(err).Msg("error reading agent file binary")
	}
	chunks := Split(binaryData)
	i := 0
	for _, chunk := range chunks {
		i++
		chunkData, err := socketio.NewEncryptedChunkedAgent(i, len(chunks), chunk, cryptor)
		if err != nil {
			errorMsg := fmt.Errorf("error decrypting agent data: %w", err)
			socket.Emit(socketio.RegisterError.ID(), socketio.SioError{
				Message: errorMsg.Error(),
				Code:    socketio.RegisterEvent.ID(),
			})
			log.Error().Str("Type", "Loader").Str("Agent.name", agentName).Str("Agent.ID", data.ID).Err(err).Msg("error encrypting agent chunk")
		}
		socket.Emit(socketio.ReceiveFatAgent.ID(), gosio.Binary(chunkData))
		// time.Sleep(5 * time.Second)
	}
}

func Split(data []byte) [][]byte {
	const chunkSize = 1024 * 1024 // 5 MB
	totalSize := len(data)
	var chunks [][]byte
	total := 0
	for i := 0; i < totalSize; i += chunkSize {
		end := i + chunkSize
		if end > totalSize {
			end = totalSize
		}
		chunk := data[i:end]
		chunks = append(chunks, chunk)
		total += len(chunk)
	}

	return chunks
}
