package socket_io

// Register is used to communicate the data when performing RegisterEvent
type Register struct {
	Id        string `json:"id"`
	Name      []byte `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

const RegisterEvent = "register"
const RegisterError = "register error"
const RegisterSuccess_AskSSHPassword = "register success"

// Deregister is used to communicate when the agent disconnects
type Deregister struct {
}

const DeregisterEvent = "deregister"
const DeregisterError = "deregister error"
const DeregisterSuccess = "deregister success"

const Connect = "connect"
const ConnectError = "connect error"
const ConnectSuccess = "connect error"

const Disconnect = "connect"
const DisconnectError = "connect error"
const DisconnectSuccess = "connect error"

type DisconnectMessage struct{}
