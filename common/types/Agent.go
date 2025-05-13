package types

import (
	"time"

	"Goauld/common/ssh"
)

type Agent struct {
	Id                   string                     `gorm:"primaryKey" json:"id" yaml:"id"`
	Name                 string                     `gorm:"type:text;unique" json:"name" yaml:"name"`
	SshMode              string                     `gorm:"type:text" json:"ssh_mode" yaml:"ssh_mode"`
	UsedPorts            string                     `gorm:"type:string" json:"usedPorts" yaml:"usedPorts"`
	RemotePortForwarding []ssh.RemotePortForwarding `gorm:"serializer:json" json:"remote_port_forwarding" yaml:"remote_port_forwarding"`
	PrivateKey           string                     `gorm:"type:text" json:"privateKey,omitempty" yaml:"privateKey,omitempty"`
	PublicKey            string                     `gorm:"type:text" json:"publicKey,omitempty" yaml:"publicKey,omitempty"`
	Source               string                     `gorm:"type:text" json:"source" yaml:"source"`
	Connected            bool                       `gorm:"type:boolean" json:"connected" yaml:"connected"`
	SharedSecret         string                     `gorm:"type:text" json:"sharedSecret,omitempty" yaml:"sharedSecret,omitempty"`
	SshPasswd            string                     `gorm:"type:text" json:"sshPasswd" yaml:"sshPasswd"`
	OneTimePassword      string                     `gorm:"type:text" json:"oneTimePassword,omitempty" yaml:"oneTimePassword"`
	LastUpdated          time.Time                  `gorm:"type:datetime" json:"lastUpdated" yaml:"lastUpdated"`
	LastPing             time.Time                  `gorm:"type:datetime" json:"lastPing" yaml:"lastPing"`

	Platform     string `gorm:"type:text" json:"platform"`
	Architecture string `gorm:"type:text" json:"architecture"`
	Username     string `gorm:"type:text" json:"username"`
	Hostname     string `gorm:"type:text" json:"hostname"`
	IPs          string `gorm:"type:text" json:"ips"`
	Path         string `gorm:"type:text" json:"path"`
}
