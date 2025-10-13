package socketio

// Register is used to communicate the data when performing RegisterEvent.
type Register struct {
	ID        string `json:"id"`
	Name      []byte `json:"name"`
	SharedKey []byte `json:"shared_key"`
}

// Deregister is used to communicate when the agent disconnects.
type Deregister struct{}

// DisconnectMessage  is used to communicate when the agent disconnects.
type DisconnectMessage struct{}
