//go:build !mini

package control

import (
	"Goauld/agent/clipboard"
	"Goauld/agent/config"
	"Goauld/common"
	"Goauld/common/crypto"
	"Goauld/common/log"
	"Goauld/common/ssh"
	"context"
	"errors"
	"fmt"
	"strings"

	socketio "Goauld/common/socket.io"

	sio "github.com/hazegard/socket.io-go"
	"golang.org/x/crypto/bcrypt"
)

func AddHandlers(socket sio.ClientSocket, cpc *ControlPlanClient) {
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
		cpc.configDone <- "Done"
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

		err = clipboard.Paste(context.Background(), []byte(message.Content))
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

		content, err := clipboard.Copy(context.Background())
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: Copy")
		}

		res := err == nil

		resp := socketio.ClipboardMessage{
			Error:   res,
			Content: string(content),
		}

		response, err := socketio.NewEncryptedClipboardMessageEventMessage(resp, config.Get().Cryptor)
		if err != nil {
			log.Error().Err(err).Msg("OnEvent: EncryptPasswordValidationRequest")
		}
		socket.Emit(req.EventID, response)
		log.Trace().Bool("Response", res).Msgf("Emit: %s", req.EventID)
	})
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
