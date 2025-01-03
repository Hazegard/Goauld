package types

import "time"

type WSHState struct {
	AgentId       string `json:"agentId,omitempty"`
	SSHLocaleAddr string `json:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `json:"SSHRemoteAddr,omitempty"`
	WSLocaleAddr  string `json:"WSLocaleAddr,omitempty"`
	WSRemoteAddr  string `json:"WSRemoteAddr,omitempty"`
}

type TLSSHState struct {
	AgentId       string `json:"agentId,omitempty"`
	SSHLocaleAddr string `json:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `json:"SSHRemoteAddr,omitempty"`
	TLSLocaleAddr string `json:"TLSLocaleAddr,omitempty"`
	TLSRemoteAddr string `json:"TLSRemoteAddr,omitempty"`
}

type SSHTTState struct {
	AgentId       string `json:"agentId,omitempty"`
	SSHLocaleAddr string `json:"SSHLocaleAddr,omitempty"`
	SSHRemoteAddr string `json:"SSHRemoteAddr,omitempty"`
}

type SocketIOState struct {
	AgentId   string `json:"agentId,omitempty"`
	SocketId  string `json:"socketId,omitempty"`
	Connected bool   `json:"connected,omitempty"`
	Recovered bool   `json:"recovered,omitempty"`
}

type State struct {
	Id           string    `json:"id,omitempty"`
	Name         string    `json:"name,omitempty"`
	SSHMode      string    `json:"SSHMode,omitempty"`
	UsedPorts    string    `json:"usedPorts,omitempty"`
	LastUpdated  time.Time `json:"lastUpdated"`
	Platform     string    `json:"platform,omitempty"`
	Architecture string    `json:"architecture,omitempty"`
	Username     string    `json:"username,omitempty"`
	Hostname     string    `json:"hostname,omitempty"`
	IPs          string    `json:"IPs,omitempty"`
	Path         string    `json:"path,omitempty"`

	TLSSH    TLSSHState    `json:"TLSSH"`
	WSSH     WSHState      `json:"WSSH"`
	SSHTTP   SSHTTState    `json:"SSHTTP"`
	SocketIO SocketIOState `json:"socketIO"`
}
