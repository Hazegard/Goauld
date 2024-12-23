package control

import (
	socketio "Goauld/common/socket.io"
	"Goauld/server/config"
	"Goauld/server/db"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	sio "github.com/karagenc/socket.io-go"
)

func RunSocketIOServer() *SocketIOServer {
	cfg := sio.ServerConfig{}
	io := sio.NewServer(&cfg)
	sio := new(SocketIOServer)

	sio.Setup(io.Of("/"))
	err := io.Run()
	if err != nil {
		log.Fatalln(err)
	}
	router := http.NewServeMux()
	router.Handle("/socket.io/", io)

	server := &http.Server{
		Addr:    config.Get().ListenAddress,
		Handler: router,

		// It is always a good practice to set timeouts.
		ReadTimeout: 120 * time.Second,
		IdleTimeout: 120 * time.Second,

		// HTTPWriteTimeout returns io.PollTimeout + 10 seconds (extra 10 seconds to write the response).
		// You should either set this timeout to 0 (infinite) or some value greater than the io.PollTimeout.
		// Otherwise poll requests may fail.
		WriteTimeout: io.HTTPWriteTimeout(),
	}
	err = server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Fatalln(err)
	}
	return nil
}

type SocketIOServer struct {
	agents   map[sio.ServerSocket]*db.Agent
	agentsMu sync.Mutex
}

func (a *SocketIOServer) AddAgent(agent *db.Agent, id sio.ServerSocket) {
	a.agentsMu.Lock()
	a.agents[id] = agent
	a.agentsMu.Unlock()
}
func (a *SocketIOServer) RemoveAgent(id sio.ServerSocket) {
	a.agentsMu.Lock()
	delete(a.agents, id)
	a.agentsMu.Unlock()
}
func (a *SocketIOServer) GetAgent(id sio.ServerSocket) *db.Agent {
	a.agentsMu.Lock()
	agent := a.agents[id]
	a.agentsMu.Unlock()
	return agent
}

func (a *SocketIOServer) Setup(root *sio.Namespace) {
	a.agents = make(map[sio.ServerSocket]*db.Agent)
	a.agentsMu = sync.Mutex{}
	root.OnConnection(func(socket sio.ServerSocket) {
		socket.OnEvent(socketio.RegisterEvent, func(data socketio.Register) {
			fmt.Println("OnEvent socketio.RegisterEvent!")
			agent, err := db.Get().FindOrCreate(data.Id)
			if err != nil {
				errorMsg := fmt.Errorf("error retrieving agent: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterError,
				})
				fmt.Println(errorMsg)
			}
			fmt.Println(agent.Id)

			sharedSecret, err := config.Get().Decrypt(data.SharedKey)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting shared secret: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				fmt.Println(errorMsg)
			}
			fmt.Println(sharedSecret)

			name, err := config.Get().Decrypt(data.Name)
			if err != nil {
				errorMsg := fmt.Errorf("error decrypting name: %s", err)
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: errorMsg.Error(),
					Code:    socketio.RegisterEvent,
				})
				fmt.Println(errorMsg)
			}
			fmt.Println(name)
			fmt.Println(name)
			fmt.Println(name)
			fmt.Println(name)
			fmt.Println(name)
			fmt.Println(name)

			agent.SetSharedSecret(sharedSecret)
			agent.SetName(name)
			agent.SetConnect()
			a.AddAgent(agent, socket)

			if agent.PrivateKey == "" {
				err := agent.InitKeys()
				if err != nil {
					socket.Emit(socketio.RegisterError, socketio.SioError{
						Message: "error generating keys",
						Code:    socketio.RegisterError,
					})
				}
			}
			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting fields",
					Code:    socketio.RegisterError,
				})
			}
			message, err := socketio.NewEncryptedSshPrivateKeyMessage(agent.PrivateKey, cryptor)
			if err != nil {
				socket.Emit(socketio.RegisterError, socketio.SioError{
					Message: "error encrypting keys",
					Code:    socketio.RegisterError,
				})
			}
			socket.Emit(socketio.SendSshPrivateKeyEvent, message)
		})

		socket.OnEvent(socketio.SendAgentSshPasswordEvent, func(data []byte) {
			fmt.Println("OnEvent socketio.RegisterEvent!")
			agent := a.GetAgent(socket)
			cryptor, err := agent.GetCryptor()
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error geting decryptor for agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
			}
			password, err := socketio.DecryptAgentSshPasswordMessage(data, cryptor)
			if err != nil {
				socket.Emit(socketio.SendAgentSshPasswordError, socketio.SioError{
					Message: "error decrypting agent ssh password",
					Code:    socketio.SendAgentSshPasswordError,
				})
			}
			agent.SetSshpassword(password.AgentSshPassword)
			fmt.Println("OnEvent socketio.SendAgentSshPasswordEvent!")
			fmt.Println("SSH password for Agent %s (%s): %s", agent.Id, agent.ID)
		})

		socket.OnEvent(socketio.DeregisterEvent, func(data socketio.Deregister) {
			agent := a.GetAgent(socket)
			agent.SetDisconnect()

		})

		socket.OnDisconnect(func(reason sio.Reason) {
			fmt.Println("OnDisconnect!")
			agent := a.GetAgent(socket)
			fmt.Println(agent.Id)
			fmt.Println(agent.Name)
			agent.SetDisconnect()
			fmt.Println("Agent Disconnected!")

		})
	})
}
