package ssh

import (
	"Goauld/common/net"
	"fmt"
	stdnet "net"
	"strconv"
	"strings"
)

type ReversePortForwarding struct {
	RemotePort int
	LocalPort  int
	LocalIP    string
}

func (rpf *ReversePortForwarding) GetRemote() string {
	return stdnet.JoinHostPort("127.0.0.1", strconv.Itoa(rpf.RemotePort))
}

func (rpf *ReversePortForwarding) GetLocal() string {
	return stdnet.JoinHostPort(rpf.LocalIP, strconv.Itoa(rpf.LocalPort))
}

func (rpf *ReversePortForwarding) String() string {
	return fmt.Sprintf("%d:%s:%d", rpf.RemotePort, rpf.LocalIP, rpf.LocalPort)
}

func (rpf *ReversePortForwarding) UnmarshalText(text []byte) error {
	// Convert text (which is a byte slice) to a string
	s := string(text)

	// Split the string into parts based on the ':' delimiter
	parts := strings.Split(s, ":")
	if len(parts) < 2 || len(parts) > 3 {
		return fmt.Errorf("invalid format, expected 'RemotePort:LocalIP:LocalPort' or 'RemotePort::LocalPort'")
	}

	// Parse RemotePort (int)
	remotePort, err := strconv.Atoi(parts[0])
	if err != nil {
		return fmt.Errorf("invalid RemotePort: %v", err)
	}

	if !net.IsValidPort(remotePort) {
		return fmt.Errorf("invalid RemotePort: %v", remotePort)
	}

	// If LocalIP is not provided, set it to the default value
	var localIP string
	if len(parts) == 3 {

		localIP = parts[1] // LocalIP is the second part when all three parts are present
	} else {
		localIP = "127.0.0.1" // Default value when LocalIP is missing
	}

	if !net.IsValidIP(localIP) {
		return fmt.Errorf("invalid LocalIP: %v", localIP)
	}

	// Parse LocalPort (int)
	localPort, err := strconv.Atoi(parts[len(parts)-1]) // The last part is always the LocalPort
	if err != nil {
		return fmt.Errorf("invalid LocalPort: %v", err)
	}
	if !net.IsValidPort(localPort) {
		return fmt.Errorf("invalid LocalPort: %v", localPort)
	}

	// Set the fields of ReversePortForwarding
	rpf.RemotePort = remotePort
	rpf.LocalIP = localIP
	rpf.LocalPort = localPort

	return nil
}
