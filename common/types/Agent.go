package types

import (
	"Goauld/common/ssh"
	"time"
)

type Agent struct {
	Id                   string                     `gorm:"primaryKey" json:"id"`
	Name                 string                     `gorm:"type:text;unique" json:"name"`
	SshMode              string                     `gorm:"type:text" json:"ssh_mode"`
	UsedPorts            string                     `gorm:"type:string" json:"usedPorts"`
	RemotePortForwarding []ssh.RemotePortForwarding `gorm:"serializer:json" json:"remote_port_forwarding"`
	PrivateKey           string                     `gorm:"type:text" json:"privateKey,omitempty"`
	PublicKey            string                     `gorm:"type:text" json:"publicKey,omitempty"`
	Source               string                     `gorm:"type:text" json:"source"`
	Connected            bool                       `gorm:"type:boolean" json:"connected"`
	SharedSecret         string                     `gorm:"type:text" json:"sharedSecret,omitempty"`
	SshPasswd            string                     `gorm:"type:text" json:"sshPasswd"`
	OneTimePassword      string                     `gorm:"type:text" json:"one_time_password,omitempty"`
	LastUpdated          time.Time                  `gorm:"type:datetime" json:"lastUpdated"`

	Platform     string `gorm:"type:text" json:"platform"`
	Architecture string `gorm:"type:text" json:"architecture"`
	Username     string `gorm:"type:text" json:"username"`
	Hostname     string `gorm:"type:text" json:"hostname"`
	IPs          string `gorm:"type:text" json:"ips"`
	Path         string `gorm:"type:text" json:"path"`
}

// // ParseFPR parse the remote port forwarded ports stored in the RemotePortForwarding field
// // And populate the Rpf field with the associated struct
// func (a *Agent) ParseFPR() {
// 	rpfs := strings.Split(a.RemotePortForwarding, ",")
// 	for _, rpf := range rpfs {
// 		_rpf := ssh.RemotePortForwarding{}
// 		err := _rpf.UnmarshalText([]byte(rpf))
// 		if err != nil {
// 			continue
// 		}
// 		a.Rpf = append(a.Rpf, _rpf)
// 	}
// }
