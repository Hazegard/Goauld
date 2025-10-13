// Package persistence holds the database
package persistence

import (
	commonnet "Goauld/common/net"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"Goauld/common/crypto"
	"Goauld/common/ssh"
	"Goauld/common/types"
	"Goauld/common/utils"

	"gorm.io/gorm"
)

// Agent is the common struct used to represent an agent.
type Agent struct {
	types.Agent

	CreatedAt time.Time      `json:"-"`
	UpdatedAt time.Time      `json:"-"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
	SocketID  string         `json:"-"`
	cryptor   *crypto.SymCryptor
}

// JSON Return a json Agent.
func (a *Agent) JSON() ([]byte, error) {
	a.CreatedAt = time.Now()

	return json.Marshal(a)
}

// GetCryptor returns the encryption class that allows to encrypt or decrypt data using the
// shared encryption key.
func (a *Agent) GetCryptor() (*crypto.SymCryptor, error) {
	if a.cryptor == nil {
		if a.SharedSecret == "" {
			return nil, errors.New("error, shared secret unavailable")
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

// NewAgent return a new Agent.
func NewAgent(source string) Agent {
	return Agent{
		Agent: types.Agent{
			Source: source,
		},
	}
}

// InitKeys initialize the public and private SSH keys for the agent.
func (a *Agent) InitKeys() error {
	priv, pub, err := ssh.GenKey()
	if err != nil {
		return err
	}
	a.PrivateKey = priv
	a.PublicKey = pub

	return nil
}

// AddPort adds the port to the array of ports used by the agent.
func (a *Agent) AddPort(port int) {
	ports := portStringToInt(a.UsedPorts)
	if len(ports) == 1 && ports[0] == 0 {
		ports = []int{}
	}
	ports = append(ports, port)

	ports = utils.Unique(ports)
	slices.Sort(ports)
	a.UsedPorts = portIntToString(ports)
}

// DeletePort remove the port from the array of ports used by the agent.
func (a *Agent) DeletePort(port int) {
	usedPorts := portStringToInt(a.UsedPorts)
	for i, p := range usedPorts {
		if p == (port) {
			// Remove the port by slicing
			usedPorts = append(usedPorts[:i], usedPorts[i+1:]...)
			a.UsedPorts = portIntToString(usedPorts)

			return // Exit after removing the port
		}
	}
	a.UsedPorts = portIntToString(usedPorts)
}

// SetRemotePortForwarding sets the remote port forwarding information.
func (a *Agent) SetRemotePortForwarding(rpf []ssh.RemotePortForwarding) {
	a.RemotePortForwarding = rpf
	// var rpfString []string
	// for _, v := range rpf {
	// 	rpfString = append(rpfString, v.Info())
	// }
	// a.RemotePortForwarding = strings.Join(rpfString, ",")
}

// SetConnect sets the agent state to "connected" state.
func (a *Agent) SetConnect() {
	a.Connected = true
	a.LastUpdated = time.Now()
}

// SetDisconnect sets the agent state to "disconnected" state.
func (a *Agent) SetDisconnect() {
	a.Connected = false
	a.LastUpdated = time.Now()
}

// SetSharedSecret set the shared encryption key to the agent.
func (a *Agent) SetSharedSecret(secret string) {
	a.SharedSecret = secret
	// a.save()
}

// SetName set the name to the agent.
func (a *Agent) SetName(name string) {
	a.Name = name
}

// SetSSHPassword set the SSH password to the agent.
func (a *Agent) SetSSHPassword(pwd string) {
	a.SSHPasswd = pwd
}

// GetForwardedPorts return the list of ports forwarded by the agent.
func (a *Agent) GetForwardedPorts() []int {
	usedPorts := portStringToInt(a.UsedPorts)
	for _, rpf := range a.RemotePortForwarding {
		usedPorts = append(usedPorts, rpf.ServerPort)
	}

	return utils.Unique(usedPorts)
}

// ValidatePasswordAndRotateIfTrue check if the incoming password matches the current password.
// If the password matches, we rotate it.
func (a *Agent) ValidatePasswordAndRotateIfTrue(password string) error {
	isValid := password == a.OneTimePassword
	if !isValid {
		return errors.New("invalid password")
	}
	newPassword, err := crypto.GeneratePassword(32)
	if err != nil {
		return err
	}
	a.OneTimePassword = newPassword

	return nil
}

// IsPortForwarded checks if the agent forwards the port.
func (a *Agent) IsPortForwarded(port int) bool {
	return utils.Contains(a.GetForwardedPorts(), port)
}

// GetAllAgents returns all the agents in the database.
func (db *DB) GetAllAgents() ([]Agent, error) {
	var agents []Agent
	result := db.db.Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}

	return agents, nil
}

// GetAllAgentsSanitized returns all the agents in the database, but clear all secrets.
func (db *DB) GetAllAgentsSanitized() ([]Agent, error) {
	var agents []Agent
	result := db.db.Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}
	for i := range agents {
		agents[i].SharedSecret = "[REDACTED]"
		agents[i].PrivateKey = "[REDACTED]"
		agents[i].PublicKey = "[REDACTED]"
		agents[i].SSHPasswd = "[REDACTED]"
		agents[i].OneTimePassword = "[REDACTED]"
	}

	return agents, nil
}

// FindAgentByID returns the agent identified by id.
func (db *DB) FindAgentByID(id string) (*Agent, error) {
	var agent Agent
	// Pass the struct as a pointer
	result := db.db.First(&agent, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}

	return &agent, nil
}

// FindAgentByName returns the agent identified by id.
func (db *DB) FindAgentByName(name string) (*Agent, error) {
	var agent Agent
	// Pass the struct as a pointer
	result := db.db.First(&agent, "name = ?", name)
	if result.Error != nil {
		return nil, result.Error
	}

	return &agent, nil
}

// FindAgentByIDAndName returns the agent identified by id.
func (db *DB) FindAgentByIDAndName(id string, name string) (*Agent, error) {
	var agent Agent
	// Pass the struct as a pointer
	result := db.db.First(&agent, "id = ? and name = ?", id, name)
	if result.Error != nil {
		return nil, result.Error
	}

	return &agent, nil
}

// ValidatePasswordAndRotateIfTrue use atomic SQL query
// to update the password if the provided one matched the stored one.
func (db *DB) ValidatePasswordAndRotateIfTrue(id string, password string) error {
	newPassword, err := crypto.GeneratePassword(32)
	if err != nil {
		return err
	}
	res := db.db.Model(&Agent{}).
		Where("id = ? and one_time_password = ?", id, password).
		Update("one_time_password", newPassword)
	if res.Error != nil {
		return fmt.Errorf("could not update agent: %w", res.Error)
	}
	if res.RowsAffected != 1 {
		return errors.New("could not update agent: no agent updated")
	}

	return nil
}

// UpdateAgentFieldShadow update the specified field information in the database without touching
// the lastUpdated field.
// Mainly used to update the last ping field.
func (db *DB) UpdateAgentFieldShadow(agent *Agent, fields ...string) error {
	result := db.db.Select(fields).Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("could not update agent: %w", result.Error)
	}

	return nil
}

// UpdateAgentField update the specified field information in the database.
func (db *DB) UpdateAgentField(agent *Agent, fields ...string) error {
	agent.LastUpdated = time.Now()
	fields = append(fields, "LastUpdated")
	agent.LastPing = time.Now()
	fields = append(fields, "LastPing")
	result := db.db.Select(fields).Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("could not update agent: %w", result.Error)
	}

	return nil
}

// UpdateAgent update the agent information in the database.
func (db *DB) UpdateAgent(agent *Agent) error {
	result := db.db.Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("could not update agent: %w", result.Error)
	}

	return nil
}

// AddPortToAgent adds the port to the UsedPorts field of the agent.
func (db *DB) AddPortToAgent(id string, port int) error {
	agent, err := db.FindAgentByID(id)
	if err != nil {
		return err
	}
	agent.AddPort(port)

	return db.UpdateAgentField(agent, "UsedPorts")
}

// RemovePortToAgent removes the port from the UsedPorts field of the agent.
func (db *DB) RemovePortToAgent(id string, port int) error {
	agent, err := db.FindAgentByID(id)
	if err != nil {
		return err
	}
	agent.DeletePort(port)

	return db.UpdateAgentField(agent, "UsedPorts")
}

// FindOrCreate retrieves the agent from the database
// If no agent corresponding to this ID exists,
// an empty one that will be populated later is returned.
func (db *DB) FindOrCreate(id string, name string) (*Agent, error) {
	agent, _ := db.FindAgentByIDAndName(id, name)
	if agent != nil {
		return agent, nil
	}
	OneTimePassword, err := crypto.GeneratePassword(32)
	if err != nil {
		return nil, err
	}
	agent = &Agent{}
	agent.ID = id
	agent.Name = name
	agent.OneTimePassword = OneTimePassword
	err = db.CreateAgent(agent)
	if err != nil {
		return nil, fmt.Errorf("could not create agent: %w", err)
	}

	return agent, nil
}

// CreateAgent creates the agent in the database.
func (db *DB) CreateAgent(agent *Agent) error {
	result := db.db.Create(agent)
	if result.Error != nil {
		return fmt.Errorf("could not create agent: %w", result.Error)
	}

	return nil
}

// DeleteAgentByID delete the agent from the database.
func (db *DB) DeleteAgentByID(id string) error {
	res := db.db.Unscoped().Delete(&Agent{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}

	return nil
}

// GetAgentsByUsedPort returns all agents using the port
// only on agent should be returned at a time.
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

// SetAgentSSHMode updates the agent to reflect the current connection mode used
// Direct (SSH), SSH over TLS, ssh over Websockets, SSH over HTTP.
func (db *DB) SetAgentSSHMode(id string, mode string, remoteAddr string) error {
	switch mode {
	case "HTTP", "SSH", "TLS", "WS", "DNS", "OFF", "QUIC":
	default:
		return fmt.Errorf("unknown mode: %s", mode)
	}
	agent, err := db.FindAgentByID(id)
	if err != nil {
		return fmt.Errorf("could not find agent: %w", err)
	}
	if agent == nil {
		return errors.New("agent not found")
	}
	if (agent.SSHMode == "OFF" || agent.SSHMode == "") && mode == "SSH" {
		// We consider that the agent connecting using a tunnel is already updated in the database
		agent.SSHMode = mode
	} else if mode != "SSH" {
		agent.SSHMode = mode
	}
	// If disconnected, no ports are used
	if mode == "OFF" {
		agent.UsedPorts = "/"
		agent.Connected = false
		agent.RemotePortForwarding = []ssh.RemotePortForwarding{}
		agent.SocketID = ""
	}

	remoteIP, _ := commonnet.ExtractIP(remoteAddr)
	isLoopback := commonnet.IsLoopback(remoteIP)
	if isLoopback {
		// If the new address is loopback, only set if none stored yet
		if agent.RemoteAddr == "" {
			agent.RemoteAddr = remoteIP
		}
	} else if commonnet.IsValidIP(remoteIP) {
		// If the new address is NOT loopback, always override
		agent.RemoteAddr = remoteIP
	}
	agent.LastUpdated = time.Now()
	err = db.UpdateAgentField(agent, "SSHMode", "LastUpdated", "UsedPorts", "Connected", "RemotePortForwarding", "SocketID", "RemoteAddr")
	if err != nil {
		return fmt.Errorf("could not update agent: %w", err)
	}

	return nil
}

// portStringToInt converts a string of port separated by a comma
// to a slice of the ports.
func portStringToInt(port string) []int {
	if port == "/" {
		return []int{}
	}
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

// portStringToInt converts a slice of the ports
// to a string of port separated by a comma.
func portIntToString(port []int) string {
	var portsString []string
	for _, p := range port {
		portsString = append(portsString, strconv.Itoa(p))
	}
	res := strings.Join(portsString, ",")
	if res == "" {
		res = "/"
	}

	return res
}
