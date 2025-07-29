package socket_io

import "strconv"

type EventType int

func (e EventType) ID() string {
	return strconv.Itoa(int(e))
}

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
	SendSshPrivateKeyEvent
	SendSshHPrivateKeyError
	SendSshPrivateKeySuccess
	PasswordValidationRequestEvent
	PasswordValidationRequestResponse
)
