package socketio

type RelayEvent struct {
	ID   string `json:"id"`
	Data []byte `json:"data"`
}
