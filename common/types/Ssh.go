package types

type ServerToAGentPassword struct {
	AgentPassword  string `json:"ap"`
	ServerPassword string `json:"sp"`
}

func NewServerToAGentPassword(agentPassword string, serverPassword string) ServerToAGentPassword {
	return ServerToAGentPassword{agentPassword, serverPassword}
}
