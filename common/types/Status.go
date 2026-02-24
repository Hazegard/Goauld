//nolint:revive
package types

import (
	"Goauld/server/config"
	"time"

	"gorm.io/gorm"
)

type Status struct {
	Version      string              `json:"version" yaml:"version"`
	ActiveAgents []State             `json:"activeAgents" yaml:"activeAgents"`
	AllAgents    []DbAgent           `json:"stoppedAgents" yaml:"stoppedAgents"`
	Config       config.ServerConfig `json:"config" yaml:"config"`
}

type DbAgent struct {
	Agent `yaml:",omitempty,inline,alias"`

	CreatedAt time.Time      `json:"created_at" yaml:"created_at"`
	UpdatedAt time.Time      `json:"updated_at" yaml:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at" yaml:"deleted_at"`
	SocketID  string         `json:"socket_id" yaml:"socket_id"`
}
