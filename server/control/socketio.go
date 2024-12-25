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
			log.Debug().Msgf("socketio.RegisterEvent (%s)!", data.Id)
			agent, err := db.Get().FindOrCreate(data.Id)

			if err != nil {
				errorMsg := fmt.Errorf("error retrieving agent: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterError,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError retriving agent(%s)", data.Id)
				return
			}

			sharedSecret, err := config.Get().Decrypt(data.SharedKey)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting shared secret (%s): %s", data.Id, err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError decrypting shared secret (%s)", data.Id)
				return
			}

			agentName, err := config.Get().Decrypt(data.Name)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting name: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError error decrypting agent name (%s)", data.Id)
				return
			}

			agent.SetSharedSecret(sharedSecret)
			agent.SetName(agentName)
			agent.SetConnect()
			sio.agentStore.SioAddAgent(agent, socket)

			if agent.PrivateKey == "" {
				err := agent.InitKeys()
				if err != nil {
					socket.Emit(socketio.RegisterError, socketio.SioError{
						Message: "error generating keys",
						Code:    socketio.RegisterError,
					})
					log.Error().Err(err).Msgf("socketio.RegisterError error generating ssh keys (%s / %s)", agentName, data.Id)
					return
				}
			}
			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting fields",
					Code:    socketio.RegisterError,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError error encrypting ssh keys (%s / %s)", agentName, data.Id)
				return
			}
			message, err := socketio.NewEncryptedSshPrivateKeyMessage(agent.PrivateKey, cryptor)
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting keys",
					Code:    socketio.RegisterError,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError error encrypting ssh keys (%s / %s)", agentName, data.Id)
				return
			}
			socket.Emit(socketio.SendSshPrivateKeyEvent, message)
			log.Debug().Msgf("socketio.SendSshPrivateKeyEvent (%s / %s)", agentName, data.Id)
		})

		socket.OnEvent(socketio.SendAgentSshPasswordEvent, func(data []byte) {
			agent := sio.agentStore.SioGetAgent(socket)
			log.Debug().Msgf("socketio.SendAgentSshPasswordEvent (%s / %s)", agent.Name, agent.Id)

			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error geting decryptor for agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError error decrypting ssh password (%s / %s)", agent.Name, agent.Id)
			}
			password, err := socketio.DecryptAgentSshPasswordMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
				log.Error().Err(err).Msgf("socketio.RegisterError error decrypting ssh password (%s / %s)", agent.Name, agent.Id)
			}
			agent.SetSshpassword(password.AgentSshPassword)
			log.Trace().Msgf("END socketio.SendAgentSshPasswordEvent (%s / %s)!", agent.Name, agent.Id)
			log.Debug().Msgf("SSH password send (%s / %s)", agent.Name, agent.Id)
		})

		socket.OnEvent(socketio.DeregisterEvent, func(data socketio.Deregister) {
			agent := sio.agentStore.SioGetAgent(socket)
			agent.SetDisconnect()
			log.Debug().Msgf("socketio.DeregisterEvent (%s / %s)!", agent.Name, agent.Id)
		})

		socket.OnDisconnect(func(reason gosio.Reason) {
			agent := sio.agentStore.SioGetAgent(socket)

			log.Debug().Msgf("socketio.Disconnect: %s / %s (%s)!", agent.Name, agent.Id, reason)
			agent.SetDisconnect()
		})
	})
}
