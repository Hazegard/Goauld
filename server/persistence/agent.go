package persistence

import (
	"Goauld/common/crypto"
	"Goauld/common/ssh"
	"Goauld/common/utils"
	"encoding/json"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type Agent struct {
	gorm.Model
	Id           string `gorm:"primaryKey" json:"id"`
	Name         string `gorm:"type:text" json:"name"`
	SshMode      string `gorm:"type:text" json:"ssh_mode"`
	UsedPorts    string `gorm:"type:string" json:"usedPorts"`
	PrivateKey   string `gorm:"type:text" json:"privateKey"`
	PublicKey    string `gorm:"type:text" json:"publicKey"`
	Source       string `gorm:"type:text" json:"source"`
	Connected    bool   `gorm:"type:boolean" json:"connected"`
	SharedSecret string `gorm:"type:text" json:"sharedSecret"`
	SshPasswd    string `gorm:"type:text" json:"sshPasswd"`
	cryptor      *crypto.SymCryptor
}

func (a *Agent) JSON() ([]byte, error) {
	return json.Marshal(a)
}

// GetCryptor returns the encryption class that allows to encryot or decrypt data using the
// shared encryption key
func (a *Agent) GetCryptor() (*crypto.SymCryptor, error) {
	if a.cryptor == nil {
		if a.SharedSecret == "" {
			return nil, fmt.Errorf("error, shared secret unavailable")
		}
		cryptor, err := crypto.NewCryptor(a.SharedSecret)
		if err != nil {
			return nil, err
		}
		a.cryptor = cryptor
		return a.cryptor, nil
	}
	return a.cryptor, nil
}

func NewAgent(source string) Agent {
	return Agent{
		Source: source,
	}
}

// InitKeys initialize the public and private SSH keys for the agent
func (a *Agent) InitKeys() error {
	priv, pub, err := ssh.GenKey()
	if err != nil {
		return err
	}
	a.PrivateKey = priv
	a.PublicKey = pub
	return nil
}

// AddPort adds the port to the array of used ports of the agent
func (a *Agent) AddPort(port int) {
	usedPorts := portStringToInt(a.UsedPorts)
	newPorts := []int{}
	// Check if the port already exists
	for _, p := range usedPorts {
		if p == port {
			return // Port already exists, do nothing
		}
	}
	// Append the port if it is unique
	newPorts = append(newPorts, port)
	a.UsedPorts = portIntToString(newPorts)
}

// DeletePort remove the port from the array of used ports of the agent
func (a *Agent) DeletePort(port int) {
	usedPorts := portStringToInt(a.UsedPorts)
	for i, p := range usedPorts {
		if p == (port) {
			// Remove the port by slicing
			usedPorts = append(usedPorts[:i], usedPorts[i+1:]...)
			return // Exit after removing the port
		}
	}
	a.UsedPorts = portIntToString(usedPorts)
}

// SetConnect sets the agent state to connected
func (a *Agent) SetConnect() {
	a.Connected = true
}

// SetDisconnect sets the agent state to disconnected
func (a *Agent) SetDisconnect() {
	a.Connected = false
}

// SetSharedSecret set the shared encryption key to the agent
func (a *Agent) SetSharedSecret(secret string) {
	a.SharedSecret = secret
	//a.save()
}

// SetSharedSecret set the name to the agent
func (a *Agent) SetName(name string) {
	a.Name = name
}

// SetSharedSecret set the SSH password to the agent
func (a *Agent) SetSshPassword(pwd string) {
	a.SshPasswd = pwd
}

// SetSSHConnectionMode update the agent to reflect the current connection mode used
// Direct (SSH), SSH over TLS, ssh over Websockets, SSH over HTTP
func (a *Agent) SetSSHConnectionMode(mode string) error {
	m := strings.ToUpper(mode)
	switch mode {
	case "HTTP", "SSH", "TLS", "WS", "DISCONNECTED":
		a.SshMode = m
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
	return nil
}

// GetAllAgents returns all the agents in the database
func (db *DB) GetAllAgents() ([]Agent, error) {
	var agents []Agent
	result := db.db.Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}
	return agents, nil
}

// FindAgent returns the agent identified by id
func (db *DB) FindAgent(id string) (*Agent, error) {
	var agent Agent
	// Pass the struct as a pointer
	result := db.db.First(&agent, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &agent, nil
}

// UpdateAgent update the agent information in the database
func (db *DB) UpdateAgent(agent *Agent) error {
	result := db.db.Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("could not update agent: %s", result.Error)
	}
	return nil
}

// AddPortToAgent adds the port to the UsedPorts field of the agent
func (db *DB) AddPortToAgent(id string, port int) error {
	agent, err := db.FindAgent(id)
	if err != nil {
		return err
	}
	agent.AddPort(port)
	return db.UpdateAgent(agent)
}

// AddPortToAgent removes the port from the UsedPorts field of the agent
func (db *DB) RemovePortToAgent(id string, port int) error {
	agent, err := db.FindAgent(id)
	if err != nil {
		return err
	}
	agent.DeletePort(port)
	return db.UpdateAgent(agent)
}

// FindOrCreate retrieves the agent from the database
// If no agent corresponding to this ID exists
// an empty one that will be populated later is returned
func (db *DB) FindOrCreate(id string) (*Agent, error) {
	agent, err := db.FindAgent(id)
	if agent != nil {
		return agent, nil
	}
	agent = &Agent{
		Id: id,
	}
	err = db.CreateAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("could not create agent: %s", err)
	}
	return agent, nil
}

// CreateAgent creates the agent in the database
func (db *DB) CreateAgent(agent *Agent) error {
	result := db.db.Create(agent)
	if result.Error != nil {
		return fmt.Errorf("could not create agent: %s", result.Error)
	}
	return nil
}

// GetAgentsByUsedPort returns all agent using the port
// only on agent should be returned at a time
func (db *DB) GetAgentsByUsedPort(port int) ([]Agent, error) {
	agents, err := db.GetAllAgents()
	if err != nil {
		return nil, err
	}
	var found []Agent
	for _, agent := range agents {
		ports := portStringToInt(agent.UsedPorts)
		if utils.Contains(ports, port) {
			found = append(found, agent)
		}
	}
	return found, nil
}

// SetAgentSshMode updates the agent to reflect the current connection mode used
// Direct (SSH), SSH over TLS, ssh over Websockets, SSH over HTTP
func (db *DB) SetAgentSshMode(id string, mode string) error {
	agent, err := db.FindAgent(id)
	if err != nil {
		return fmt.Errorf("could not find agent: %s", err)
	}
	if agent == nil {
		return fmt.Errorf("agent not found")
	}
	err = agent.SetSSHConnectionMode(mode)
	if err != nil {
		return fmt.Errorf("could not set ssh connection mode: %s", err)
	}
	err = db.UpdateAgent(agent)
	if err != nil {
		return fmt.Errorf("could not update agent: %s", err)
	}
	return nil
}

// portStringToInt converts a string of port separated by a comma
// to a slice of the ports
func portStringToInt(port string) []int {
	var ports []int
	strPorts := strings.Split(port, ",")
	for _, p := range strPorts {
		port, err := strconv.Atoi(p)
		if err != nil {
			continue
		}
		ports = append(ports, port)
	}
	return ports
}

// portStringToInt converts  a slice of the ports
// to a string of port separated by a comma
func portIntToString(port []int) string {
	portsString := []string{}
	for _, p := range port {
		portsString = append(portsString, strconv.Itoa(p))
	}
	return strings.Join(portsString, ",")
}
