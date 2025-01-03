package ssh

import (
	"Goauld/common/net"
	"fmt"
	stdnet "net"
	"strconv"
	"strings"
)

type RemotePortForwarding struct {
	ServerPort int    `json:"serverPort,omitempty"`
	AgentPort  int    `json:"agentPort,omitempty"`
	AgentIP    string `json:"agentIP,omitempty"`
	Tag        string `json:"tag,omitempty"`
}

func (rpf *RemotePortForwarding) GetRemote() string {
	return stdnet.JoinHostPort("127.0.0.1", strconv.Itoa(rpf.ServerPort))
}

func (rpf *RemotePortForwarding) GetLocal() string {
	return stdnet.JoinHostPort(rpf.AgentIP, strconv.Itoa(rpf.AgentPort))
}

func (rpf *RemotePortForwarding) String() string {
	return fmt.Sprintf("%d:%s:%d", rpf.ServerPort, rpf.AgentIP, rpf.AgentPort)
}

func (rpf *RemotePortForwarding) Info() string {
	return fmt.Sprintf("%d:%s:%d#%s", rpf.ServerPort, rpf.AgentIP, rpf.AgentPort, rpf.Tag)
}

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
