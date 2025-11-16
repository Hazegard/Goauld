//nolint:revive
package types

import "time"

// WSSHState represents the state of the SSH over Websocket connections between the agents and the server.
type WSSHState struct {
	AgentID string `yaml:"agentId,omitempty"`
	SSHConn Conn   `yaml:"sshConn,omitempty"`
	WSConn  Conn   `yaml:"wsConn,omitempty"`
}

// SSHConnection represents the state of the direct SSH  connections between the agents and the server.
type SSHConnection struct {
	AgentID       string `yaml:"agentId,omitempty"`
	SSHConn       Conn   `yaml:"sshConn,omitempty"`
	ClientVersion string `yaml:"clientVersion,omitempty"`
	SessionID     string `yaml:"sessionID,omitempty"`
	ServerVersion string `yaml:"serverVersion,omitempty"`
}

// SSHState represents the state of the SSH connections between the agents and the server.
type SSHState struct {
	AgentID       string          `yaml:"agentId,omitempty"`
	SSHConnection []SSHConnection `yaml:"SSHConnection,omitempty"`
	SSHListeners  []string        `yaml:"SSHListeners,omitempty"`
}

// TLSSHState represents the state of the SSH over TLS connections between the agents and the server.
type TLSSHState struct {
	AgentID string `yaml:"agentId,omitempty"`
	SSHConn Conn   `yaml:"sshConn,omitempty"`
	TLSConn Conn   `yaml:"tlsConn,omitempty"`
}

// QUICState represents the state of the SSH over DNQUICS connections between the agents and the server.
type QUICState struct {
	AgentID  string `yaml:"agentId,omitempty"`
	SSHConn  Conn   `yaml:"sshConn,omitempty"`
	QuicConn Conn   `yaml:"tlsConn,omitempty"`
}

// SSHTTPState represents the state of the SSH over HTTP connections between the agents and the server.
type SSHTTPState struct {
	AgentID    string `yaml:"agentId,omitempty"`
	SSHConn    Conn   `yaml:"sshConn,omitempty"`
	StreamConn Conn   `yaml:"streamConn,omitempty"`
	StreamID   uint32 `yaml:"streamID,omitempty"`
}

// SSHCDNState represents the state of the SSH over HTTP connections between the agents and the server.
type SSHCDNState struct {
	AgentID string `yaml:"agentId,omitempty"`
	SSHConn Conn   `yaml:"sshConn,omitempty"`
}

// DNSSHState represents the state of the SSH over DNS connections between the agents and the server.
type DNSSHState struct {
	AgentID      string `yaml:"agentId,omitempty"`
	UpstreamConn []Conn `yaml:"upstreamConn,omitempty"`
	KCPAddr      string `yaml:"KCPAddr,omitempty"`
	MuxSession   Conn   `yaml:"muxSession,omitempty"`
}

// Conn abstracts a connection representation.
type Conn struct {
	LocaleAddr string `yaml:"localAddr,omitempty"`
	RemoteAddr string `yaml:"remoteAddr,omitempty"`
}

// SocketIOState represents the state of the socket.io connections between the agents and the server.
type SocketIOState struct {
	AgentID   string `yaml:"agentId,omitempty"`
	SocketID  string `yaml:"socketId,omitempty"`
	Connected bool   `yaml:"connected,omitempty"`
	Recovered bool   `yaml:"recovered,omitempty"`
}

// State represents the global State.
type State struct {
	ID           string        `json:"id,omitempty" yaml:"id,omitempty"`
	Name         string        `json:"name,omitempty" yaml:"name,omitempty"`
	SSHMode      string        `json:"SSHMode,omitempty" yaml:"SSHMode,omitempty"`
	UsedPorts    string        `json:"usedPorts,omitempty" yaml:"usedPorts,omitempty"`
	LastUpdated  time.Time     `json:"lastUpdated" yaml:"lastUpdated"`
	LastPing     time.Time     `json:"lastPing" yaml:"lastPing"`
	Platform     string        `json:"platform,omitempty" yaml:"platform,omitempty"`
	Architecture string        `json:"architecture,omitempty" yaml:"architecture,omitempty"`
	Username     string        `json:"username,omitempty" yaml:"username,omitempty"`
	Hostname     string        `json:"hostname,omitempty" yaml:"hostname,omitempty"`
	IPs          string        `json:"IPs,omitempty" yaml:"IPs,omitempty"`
	Path         string        `json:"path,omitempty" yaml:"path,omitempty"`
	RemoteAddr   string        `json:"remoteAddr,omitempty" yaml:"remoteAddr,omitempty"`
	TLSSH        TLSSHState    `json:"TLSSH" yaml:"TLSSH"`
	QUIC         QUICState     `json:"QUIC" yaml:"QUIC"`
	WSSH         WSSHState     `json:"WSSH" yaml:"WSSH"`
	SSHTTP       SSHTTPState   `json:"SSHTTP" yaml:"SSHTTP"`
	SocketIO     SocketIOState `json:"socketIO" yaml:"socketIO"`
	SSH          SSHState      `json:"ssh" yaml:"ssh"`
	DNS          DNSSHState    `json:"dns" yaml:"dns"`
	CDN          SSHCDNState   `json:"cdn" yaml:"cdn"`
}
