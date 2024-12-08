package structs

import (
	"gorm.io/gorm"
)

type AgentStore struct {
	db gorm.DB
}

func (a *AgentStore) Get(key string) Agent {
	var agent Agent
	a.db.First(&agent, "id = ?", key)
	return agent
}

func (a *AgentStore) Set(agent Agent) {
	a.db.Create(agent)
}
