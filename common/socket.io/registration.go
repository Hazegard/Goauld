package socket_io

// Register is used to communicate the data when performing RegisterEvent
type Register struct {
	Id        string `json:"id"`
	Name      []byte `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

// Deregister is used to communicate when the agent disconnects
type Deregister struct{}

type DisconnectMessage struct{}
