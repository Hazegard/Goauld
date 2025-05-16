package types

import "time"

type WSHState struct {
	AgentId string `yaml:"agentId,omitempty"`
	SshConn Conn   `yaml:"sshConn,omitempty"`
	WsConn  Conn   `yaml:"wsConn,omitempty"`
}

type SSHConnection struct {
	AgentId       string `yaml:"agentId,omitempty"`
	SshConn       Conn   `yaml:"sshConn,omitempty"`
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
	AgentId string `yaml:"agentId,omitempty"`
	SshConn Conn   `yaml:"sshConn,omitempty"`
	TlsConn Conn   `yaml:"tlsConn,omitempty"`
}

type QUICState struct {
	AgentId  string `yaml:"agentId,omitempty"`
	SshConn  Conn   `yaml:"sshConn,omitempty"`
	QuicConn Conn   `yaml:"tlsConn,omitempty"`
}

type SSHTTState struct {
	AgentId    string `yaml:"agentId,omitempty"`
	SshConn    Conn   `yaml:"sshConn,omitempty"`
	StreamConn Conn   `yaml:"streamConn,omitempty"`
	StreamId   uint32 `yaml:"streamID,omitempty"`
}

type DNSSHState struct {
	AgentId      string `yaml:"agentId,omitempty"`
	UpstreamConn []Conn `yaml:"upstreamConn,omitempty"`
	KCPAddr      string `yaml:"KCPAddr,omitempty"`
	MuxSession   Conn   `yaml:"muxSession,omitempty"`
}

type Conn struct {
	LocaleAddr string `yaml:"localAddr,omitempty"`
	RemoteAddr string `yaml:"remoteAddr,omitempty"`
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
	LastPing     time.Time `yaml:"lastPing"`
	Platform     string    `yaml:"platform,omitempty"`
	Architecture string    `yaml:"architecture,omitempty"`
	Username     string    `yaml:"username,omitempty"`
	Hostname     string    `yaml:"hostname,omitempty"`
	IPs          string    `yaml:"IPs,omitempty"`
	Path         string    `yaml:"path,omitempty"`
	RemoteAddr   string    `yaml:"remoteAddr,omitempty"`

	TLSSH    TLSSHState    `yaml:"TLSSH"`
	QUIC     QUICState     `yaml:"QUIC"`
	WSSH     WSHState      `yaml:"WSSH"`
	SSHTTP   SSHTTState    `yaml:"SSHTTP"`
	SocketIO SocketIOState `yaml:"socketIO"`
	SSH      SSHState      `yaml:"ssh"`
	DNS      DNSSHState    `yaml:"dns"`
}
