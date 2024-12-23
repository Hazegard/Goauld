package db

import (
	"Goauld/common/crypto"
	"Goauld/common/ssh"
	"fmt"
	"gorm.io/gorm"
)

type Agent struct {
	gorm.Model
	Id           string `gorm:"primaryKey"`
	Name         string `gorm:"type:text"`
	UsedPorts    []int  `gorm:"type:integer[]"`
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
	a.save()
	return nil
}

func (a *Agent) save() {
	Get().UpdateAgent(a)
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

func (a *Agent) addKeys(privateKey string, publicKey string) {
	a.PrivateKey = privateKey
	a.PublicKey = publicKey
	a.save()
}

func (a *Agent) SetSharedSecret(secret string) {
	a.SharedSecret = secret
	a.save()
}
func (a *Agent) SetName(name string) {
	a.Name = name
	a.save()
}

func (a *Agent) SetSshpassword(pwd string) {
	a.SshPasswd = pwd
	a.save()
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
