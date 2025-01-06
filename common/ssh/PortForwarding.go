package ssh

import (
	"Goauld/common/net"
	"encoding/json"
	"fmt"
	stdnet "net"
	"strconv"
	"strings"
)

// RemotePortForwarding holds the port forwarding information
type RemotePortForwarding struct {
	ServerPort int    `json:"serverPort,omitempty"`
	AgentPort  int    `json:"agentPort,omitempty"`
	AgentIP    string `json:"agentIP,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// internalRemotePortForwarding is a struct used to unmarshal RemotePortForwarding as JSON
// Given that we are required to have UnmarshalText to populate the struct from the kong CLI arguments
type internalRemotePortForwarding struct {
	ServerPort int    `json:"serverPort,omitempty"`
	AgentPort  int    `json:"agentPort,omitempty"`
	AgentIP    string `json:"agentIP,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

// GetRemote returns the remote address
func (rpf *RemotePortForwarding) GetRemote() string {
	return stdnet.JoinHostPort("127.0.0.1", strconv.Itoa(rpf.ServerPort))
}

// GetLocal returns the local address
func (rpf *RemotePortForwarding) GetLocal() string {
	return stdnet.JoinHostPort(rpf.AgentIP, strconv.Itoa(rpf.AgentPort))
}

// String returns the forwarding ports using the SSH -R scheme
func (rpf *RemotePortForwarding) String() string {
	return fmt.Sprintf("%d:%s:%d", rpf.ServerPort, rpf.AgentIP, rpf.AgentPort)
}

// Info returns the string marshalled structure to be stored in the database
func (rpf *RemotePortForwarding) Info() string {
	return fmt.Sprintf("%d:%s:%d#%s", rpf.ServerPort, rpf.AgentIP, rpf.AgentPort, rpf.Tag)
}

func (rpf *RemotePortForwarding) UnmarshalJSON(data []byte) error {
	tmp := internalRemotePortForwarding{}
	err := json.Unmarshal(data, &tmp)
	if err != nil {
		return err
	}
	rpf.ServerPort = tmp.ServerPort
	rpf.AgentPort = tmp.AgentPort
	rpf.AgentIP = tmp.AgentIP
	rpf.Tag = tmp.Tag
	return nil
}

// UnmarshalText returns the struct from the string representation
func (rpf *RemotePortForwarding) UnmarshalText(text []byte) error {
	// Convert text (which is a byte slice) to a string
	s := string(text)
	tag := ""
	if s == "/" {
		return nil
	}
	// Split the string into parts based on the ':' delimiter
	parts := strings.Split(s, "#")
	if len(parts) == 2 {
		tag = parts[1]
	}
	if len(parts) > 2 {
		return fmt.Errorf("tag format: %s", s)
	}
	parts = strings.Split(parts[0], ":")
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("invalid format, expected 'ServerPort:AgentIP:AgentPort' or 'ServerPort::AgentPort'")
	}

	// Parse ServerPort (int)
	remotePort, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid ServerPort: %v", err)
	}

	if !net.IsValidPort(remotePort) {
		return fmt.Errorf("invalid ServerPort: %v", remotePort)
	}

	// If AgentIP is not provided, set it to the default value
	var localIP string
	if len(parts) == 3 {

		localIP = parts[1] // AgentIP is the second part when all three parts are present
	} else {
		localIP = "127.0.0.1" // Default value when AgentIP is missing
	}

	if !net.IsValidIP(localIP) {
		return fmt.Errorf("invalid AgentIP: %v", localIP)
	}

	// Parse AgentPort (int)
	localPort, err := strconv.Atoi(parts[len(parts)-1]) // The last part is always the AgentPort
	if err != nil {
		return fmt.Errorf("invalid AgentPort: %v", err)
	}
	if !net.IsValidPort(localPort) {
		return fmt.Errorf("invalid AgentPort: %v", localPort)
	}

	// Set the fields of RemotePortForwarding
	rpf.ServerPort = remotePort
	rpf.AgentIP = localIP
	rpf.AgentPort = localPort
	rpf.Tag = tag

	return nil
}
