package persistence

import (
	"Goauld/common/crypto"
	"Goauld/common/ssh"
	"Goauld/common/utils"
	"fmt"
	"gorm.io/gorm"
	"strconv"
	"strings"
)

type Agent struct {
	gorm.Model
	Id           string `gorm:"primaryKey"`
	Name         string `gorm:"type:text"`
	UsedPorts    string `gorm:"type:string"`
	PrivateKey   string `gorm:"type:text"`
	PublicKey    string `gorm:"type:text"`
	Source       string `gorm:"type:text"`
	Connected    bool   `gorm:"type:boolean"`
	SharedSecret string `gorm:"type:text"`
	SshPasswd    string `gorm:"type:text"`
	cryptor      *crypto.SymCryptor
}

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

func (a *Agent) InitKeys() error {
	priv, pub, err := ssh.GenKey()
	if err != nil {
		return err
	}
	a.PrivateKey = priv
	a.PublicKey = pub
	return nil
}

func (a *Agent) caca() {
	//err := db.UpdateAgent(a)
	//if err != nil {
	//	log.Error().Err(err).Msgf("error saving agent (%s): %v", a.Id, err)
	//}
}

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

func (a *Agent) SetConnect() {
	a.Connected = true
}

func (a *Agent) SetDisconnect() {
	a.Connected = false
}

func (a *Agent) addKeys(privateKey string, publicKey string) {
	a.PrivateKey = privateKey
	a.PublicKey = publicKey
	//a.save()
}

func (a *Agent) SetSharedSecret(secret string) {
	a.SharedSecret = secret
	//a.save()
}
func (a *Agent) SetName(name string) {
	a.Name = name
}

func (a *Agent) SetSshPassword(pwd string) {
	a.SshPasswd = pwd
}

func (db *DB) GetAllAgents() ([]Agent, error) {
	agents := []Agent{}
	result := db.db.Find(&agents)
	if result.Error != nil {
		return nil, result.Error
	}
	return agents, nil
}

func (db *DB) FindAgent(id string) (*Agent, error) {
	var agent Agent
	// Pass the struct as a pointer
	result := db.db.First(&agent, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}
	return &agent, nil
}

func (db *DB) UpdateAgent(agent *Agent) error {
	result := db.db.Updates(agent)
	if result.Error != nil {
		return fmt.Errorf("could not update agent: %s", result.Error)
	}
	return nil
}

func (db *DB) AddPortToAgent(id string, port int) error {
	agent, err := db.FindAgent(id)
	if err != nil {
		return err
	}
	agent.AddPort(port)
	return db.UpdateAgent(agent)
}
func (db *DB) RemovePortToAgent(id string, port int) error {
	agent, err := db.FindAgent(id)
	if err != nil {
		return err
	}
	agent.DeletePort(port)
	return db.UpdateAgent(agent)
}

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

func (db *DB) CreateAgent(agent *Agent) error {
	result := db.db.Create(agent)
	if result.Error != nil {
		return fmt.Errorf("could not create agent: %s", result.Error)
	}
	return nil
}

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

func portIntToString(port []int) string {
	portsString := []string{}
	for _, p := range port {
		portsString = append(portsString, strconv.Itoa(p))
	}
	return strings.Join(portsString, ",")
}
