package control

import (
	"Goauld/common/log"
	socketio "Goauld/common/socket.io"
	"Goauld/server/config"
	"Goauld/server/db"
	"Goauld/server/store"
	"fmt"
	gosio "github.com/karagenc/socket.io-go"
)

type SocketIO struct {
	agentStore *store.AgentStore
	server     *gosio.Server
}

func InitSocketIOServer(agentStore *store.AgentStore) *gosio.Server {

	io := gosio.NewServer(&gosio.ServerConfig{})
	socketIO := &SocketIO{
		agentStore: agentStore,
	}
	socketIO.Setup(io.Of("/"))
	err := io.Run()
	if err != nil {
	}

	return io
}

func (sio *SocketIO) Setup(root *gosio.Namespace) {
	root.OnConnection(func(socket gosio.ServerSocket) {
		socket.OnEvent(socketio.RegisterEvent, func(data socketio.Register) {
			log.Debug().Str("Agent.Id", data.Id).Msg("socketio.RegisterEvent")
			agent, err := db.Get().FindOrCreate(data.Id)
			if err != nil {
				errorMsg := fmt.Errorf("error retrieving agent: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterError,
				})
				log.Error().Str("Agent.name", "").Str("Agent.Id", data.Id).Err(err).Msg("socketio.RegisterError retriving agent")
				return
			}

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

			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.decrypting agent name")
			agentName, err := config.Get().Decrypt(data.Name)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting name: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error decrypting agent name")
				return
			}

			agent.SetSharedSecret(sharedSecret)
			agent.SetName(agentName)
			agent.SetConnect()
			sio.agentStore.SioAddAgent(agent, socket)

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
			}

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

			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("START socketio.RegisterEvent sending encrypted ssh private key")
			socket.Emit(socketio.SendSshPrivateKeyEvent, message)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.SendSshPrivateKeyEvent")
		})

		socket.OnEvent(socketio.SendAgentSshPasswordEvent, func(data []byte) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.SendAgentSshPasswordEvent")

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error geting decryptor for agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error decrypting ssh password")
				return
			}
			password, err := socketio.DecryptAgentSshPasswordMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.RegisterError error decrypting ssh password")
				return
			}
			agent.SetSshpassword(password.AgentSshPassword)
			socket.Emit(socketio.SendAgentSshPasswordSuccess)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("END socketio.SendAgentSshPasswordEvent")
		})

		socket.OnEvent(socketio.DeregisterEvent, func(data socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			agent.SetDisconnect()
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("socketio.DeregisterEvent")
		})

		socket.OnEvent(socketio.PingEvent, func(data socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("Received socketio.PingEvent")
			socket.Emit(socketio.PongEvent)
			log.Trace().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msg("Sent socketio.PongEvent")
		})

		socket.OnDisconnect(func(reason gosio.Reason) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Msgf("socketio.Disconnect: %s", reason)

			err := sio.agentStore.CloseAgentConnections(agent.Id)
			if err != nil {
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.Disconnect: error closing agent")
			}
			sio.agentStore.SioRemoveAgent(socket)
			if err != nil {
				log.Error().Str("Agent.name", agent.Name).Str("Agent.Id", agent.Id).Err(err).Msg("socketio.Disconnect: agent disconnect failed")
			}
			agent.SetDisconnect()

		})
	})
}
