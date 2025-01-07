package transport

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"

	"Goauld/agent/agent"

	"golang.org/x/crypto/ssh"
)

const SSHD_BANNER = "SSH-2.0-"

// CheckSSHService connects to the given address, verifies that the service is SSH,
// and checks if the returned banner matches the expected one.
func CheckDirectSshAccess(address string) error {
	// Set a timeout for the connection
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()

	// Set a deadline for the read operation
	err = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		return fmt.Errorf("failed to set read deadline: %w", err)
	}

	// Read the banner line
	reader := bufio.NewReader(conn)
	banner, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("failed to read banner: %w", err)
	}

	// Trim any extra whitespace or newlines from the banner
	banner = strings.TrimSpace(banner)

	// Check if the banner matches the expected one
	if !strings.Contains(banner, SSHD_BANNER) {
		return fmt.Errorf("unexpected banner: got %s, want %s*", banner, SSHD_BANNER)
	}

	return nil
}

// DirectSshConnect perform direct SSH connection to the server
func DirectSshConnect(sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	err := CheckDirectSshAccess(agent.Get().ControlSshServer())
	if err != nil {
		return nil, fmt.Errorf("unable to access the ssh server directly (%s): %w", agent.Get().ControlSshServer(), err)
	}
	client, err := ssh.Dial("tcp", agent.Get().ControlSshServer(), sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", agent.Get().ControlSshServer(), err)
	}
	return client, nil
}
