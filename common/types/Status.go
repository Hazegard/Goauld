package types

import (
	"Goauld/server/config"
	"gorm.io/gorm"
	"time"
)

type Status struct {
	Version      string              `yaml:"version"`
	ActiveAgents []State             `yaml:"activeAgents"`
	AllAgents    []DbAgent           `yaml:"stoppedAgents"`
	Config       config.ServerConfig `yaml:"config"`
}

type DbAgent struct {
	CreatedAt time.Time      `yaml:"created_at"`
	UpdatedAt time.Time      `yaml:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" yaml:"deleted_at"`
	SocketId  string         `yaml:"socket_id"`
	Agent     `yaml:",omitempty,inline,alias"`
}
