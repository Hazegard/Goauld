//nolint:revive
package types

import (
	"Goauld/server/config"
	"time"

	"gorm.io/gorm"
)

type Status struct {
	Version      string              `yaml:"version"`
	ActiveAgents []State             `yaml:"activeAgents"`
	AllAgents    []DbAgent           `yaml:"stoppedAgents"`
	Config       config.ServerConfig `yaml:"config"`
}

type DbAgent struct {
	Agent `yaml:",omitempty,inline"`

	CreatedAt time.Time      `yaml:"created_at"`
	UpdatedAt time.Time      `yaml:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" yaml:"deleted_at"`
	SocketID  string         `yaml:"socket_id"`
}
