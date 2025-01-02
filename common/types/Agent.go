package types

import "time"

type Agent struct {
	Id           string    `gorm:"primaryKey" json:"id"`
	Name         string    `gorm:"type:text;unique" json:"name"`
	SshMode      string    `gorm:"type:text" json:"ssh_mode"`
	UsedPorts    string    `gorm:"type:string" json:"usedPorts"`
	PrivateKey   string    `gorm:"type:text" json:"privateKey,omitempty"`
	PublicKey    string    `gorm:"type:text" json:"publicKey,omitempty"`
	Source       string    `gorm:"type:text" json:"source"`
	Connected    bool      `gorm:"type:boolean" json:"connected"`
	SharedSecret string    `gorm:"type:text" json:"sharedSecret,omitempty"`
	SshPasswd    string    `gorm:"type:text" json:"sshPasswd"`
	LastUpdated  time.Time `gorm:"type:datetime" json:"lastUpdated"`

	Platform     string `gorm:"type:text" json:"platform"`
	Architecture string `gorm:"type:text" json:"architecture"`
	Username     string `gorm:"type:text" json:"username"`
	Hostname     string `gorm:"type:text" json:"hostname"`
	IPs          string `gorm:"type:text" json:"ips"`
	Path         string `gorm:"type:text" json:"path"`
}
