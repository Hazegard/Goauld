package structs

import (
	"Goauld/server/persistence"
	"gorm.io/gorm"
)

type Agent struct {
	gorm.Model
	Key       string `gorm:"primaryKey"`
	UsedPorts []int
	Password  string
	Source    string
	Connected bool
}

func NewAgent(source string) Agent {
	return Agent{
		Source: source,
	}
}

func (a *Agent) save() {
	persistence.Get().Create(a)
}

func (a *Agent) AddPort(port int) {
	// Check if the port already exists
	for _, p := range a.UsedPorts {
		if p == port {
			return // Port already exists, do nothing
		}
	}
	// Append the port if it is unique
	a.UsedPorts = append(a.UsedPorts, port)
	a.save()
}

func (a *Agent) DeletePort(port int) {
	for i, p := range a.UsedPorts {
		if p == port {
			// Remove the port by slicing
			a.UsedPorts = append(a.UsedPorts[:i], a.UsedPorts[i+1:]...)
			return // Exit after removing the port
		}
	}
	a.save()
}

func (a *Agent) SetConnect() {
	a.Connected = true
	a.save()
}

func (a *Agent) SetDisconnect() {
	a.Connected = false
	a.save()
}
