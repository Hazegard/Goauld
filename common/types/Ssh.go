package types

type ServerToAgentPassword struct {
	HashAgentPassword string `json:"hap"`
	ServerPassword    string `json:"sp"`
}

func NewServerToAGentPassword(agentPassword string, serverPassword string) ServerToAgentPassword {
	return ServerToAgentPassword{agentPassword, serverPassword}
}
