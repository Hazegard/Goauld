package socket_io

// Register is used to communicate the data when performing RegisterEvent
type Register struct {
	Id        string `json:"id"`
	Name      []byte `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

const (
	RegisterEvent                  = "register"
	RegisterError                  = "register error"
	RegisterSuccess_AskSSHPassword = "register success"
)

// Deregister is used to communicate when the agent disconnects
type Deregister struct{}

const (
	DeregisterEvent   = "deregister"
	DeregisterError   = "deregister error"
	DeregisterSuccess = "deregister success"
)

const (
	Connect        = "connect"
	ConnectError   = "connect error"
	ConnectSuccess = "connect error"
)

const (
	Disconnect        = "SioDisconnect"
	DisconnectError   = "SioDisconnect error"
	DisconnectSuccess = "SioDisconnect error"
)

type DisconnectMessage struct{}
