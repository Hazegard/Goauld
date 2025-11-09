package socketio

import "strconv"

// EventType holds the representation of a socket.IO event.
type EventType int

// ID returns the ID of an event.
func (e EventType) ID() string {
	return strconv.Itoa(int(e))
}

// All events used to communicate between the agent and the server.
const (
	SendAgentDataEvent EventType = iota
	SendAgentDataError
	SendAgentDataSuccess
	ExitEvent
	ExitError
	ExitSuccess
	AlreadyConnectedEvent
	AlreadyConnectedError
	AlreadyConnectedSuccess
	PingEvent
	PingError
	PingSuccess
	PongEvent
	PongError
	PongSuccess
	RegisterEvent
	RegisterError
	RegisterSuccessAskSSHPassword
	VersionEvent
	DeregisterEvent
	DeregisterError
	DeregisterSuccess
	Connect
	ConnectError
	ConnectSuccess
	Disconnect
	DisconnectError
	DisconnectSuccess
	SendRemotePortForwardingDataEvent
	SendRemotePortForwardingDataError
	SendRemotePortForwardingDataSuccess
	SendSSHPrivateKeyEvent
	SendSSHHPrivateKeyError
	SendSSHPrivateKeySuccess
	PasswordValidationRequestEvent
	PasswordValidationRequestResponse
	ClipboardContentEvent
	CopyClipboardRequestEvent
	ReceiveFatAgent
	WireguardPeer
	PasswordValidationRequestEventRelay
	CopyClipboardRequestEventRelay
)
