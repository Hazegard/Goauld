package types

import "time"

type WSHState struct {
	AgentId       string `yaml:"agentId,omitempty"`
	SSHLocaleAddr string `yaml:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `yaml:"SSHRemoteAddr,omitempty"`
	WSLocaleAddr  string `yaml:"WSLocaleAddr,omitempty"`
	WSRemoteAddr  string `yaml:"WSRemoteAddr,omitempty"`
}

type SSHConnection struct {
	AgentId       string `yaml:"agentId,omitempty"`
	SSHLocaleAddr string `yaml:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `yaml:"SSHRemoteAddr,omitempty"`
	ClientVersion string `yaml:"clientVersion,omitempty"`
	SessionID     string `yaml:"sessionID,omitempty"`
	ServerVersion string `yaml:"serverVersion,omitempty"`
}

type SSHState struct {
	AgentId       string          `yaml:"agentId,omitempty"`
	SSHConnection []SSHConnection `yaml:"SSHConnection,omitempty"`
	SSHListeners  []string        `yaml:"SSHListeners,omitempty"`
}

type TLSSHState struct {
	AgentId       string `yaml:"agentId,omitempty"`
	SSHLocaleAddr string `yaml:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `yaml:"SSHRemoteAddr,omitempty"`
	TLSLocaleAddr string `yaml:"TLSLocaleAddr,omitempty"`
	TLSRemoteAddr string `yaml:"TLSRemoteAddr,omitempty"`
}

type SSHTTState struct {
	AgentId       string `yaml:"agentId,omitempty"`
	SSHLocaleAddr string `yaml:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `yaml:"SSHRemoteAddr,omitempty"`
}

type DNSSHState struct {
	AgentId              string `yaml:"agentId,omitempty"`
	SSHLocaleAddr        string `yaml:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr        string `yaml:"SSHRemoteAddr,omitempty"`
	KCPAddr              string `yaml:"KCPAddr,omitempty"`
	MuxSessionLocaleAddr string `yaml:"MuxSessionLocaleAddr,omitempty"`
	MuxSessionRemoteAddr string `yaml:"MuxSessionRemoteAddr,omitempty"`
}

type SocketIOState struct {
	AgentId   string `yaml:"agentId,omitempty"`
	SocketId  string `yaml:"socketId,omitempty"`
	Connected bool   `yaml:"connected,omitempty"`
	Recovered bool   `yaml:"recovered,omitempty"`
}

type State struct {
	Id           string    `yaml:"id,omitempty"`
	Name         string    `yaml:"name,omitempty"`
	SSHMode      string    `yaml:"SSHMode,omitempty"`
	UsedPorts    string    `yaml:"usedPorts,omitempty"`
	LastUpdated  time.Time `yaml:"lastUpdated"`
	Platform     string    `yaml:"platform,omitempty"`
	Architecture string    `yaml:"architecture,omitempty"`
	Username     string    `yaml:"username,omitempty"`
	Hostname     string    `yaml:"hostname,omitempty"`
	IPs          string    `yaml:"IPs,omitempty"`
	Path         string    `yaml:"path,omitempty"`

	TLSSH    TLSSHState    `yaml:"TLSSH"`
	WSSH     WSHState      `yaml:"WSSH"`
	SSHTTP   SSHTTState    `yaml:"SSHTTP"`
	SocketIO SocketIOState `yaml:"socketIO"`
	SSH      SSHState      `yaml:"ssh"`
	DNS      DNSSHState    `yaml:"dns"`
}
