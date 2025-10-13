//nolint:revive
package types

// ServerToAgentPassword contains the password used by the client to authenticate to the server.
type ServerToAgentPassword struct {
	HashAgentPassword string `json:"hap"`
	ServerPassword    string `json:"sp"`
}

// NewServerToAGentPassword returns the password used by the client to authenticate to the server.
func NewServerToAGentPassword(agentPassword string, serverPassword string) ServerToAgentPassword {
	return ServerToAgentPassword{agentPassword, serverPassword}
}
