// Package transport holds the SSH tunneling
package transport

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"strings"
	"time"

	"Goauld/agent/config"

	"golang.org/x/crypto/ssh"
)

// SSHDBanner the SSHD banner.
const SSHDBanner = "SSH-2.0-"

// CheckDirectSSHAccess connects to the given address, verifies that the service is SSH,
// and checks if the returned banner matches the expected one.
func CheckDirectSSHAccess(address string) error {
	// Set a timeout for the connection
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	//nolint:errcheck
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
	if !strings.Contains(banner, SSHDBanner) {
		return fmt.Errorf("unexpected banner: got %s, want %s*", banner, SSHDBanner)
	}

	return nil
}

// DirectSSHConnect performs a direct SSH connection to the server
// and will abort dialing or handshaking if ctx is cancelled.
func DirectSSHConnect(ctx context.Context, sshConfig *ssh.ClientConfig) (*ssh.Client, error) {
	addr := config.Get().ControlSSHServer()

	if err := CheckDirectSSHAccess(addr); err != nil {
		return nil, fmt.Errorf(
			"unable to access the SSH server directly (%s): %w",
			addr, err,
		)
	}

	// 2) Dial TCP with context
	dialer := &net.Dialer{}
	rawConn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %s: %w", addr, err)
	}

	// If the context has a deadline, use it to bound the handshake
	if dl, ok := ctx.Deadline(); ok {
		_ = rawConn.SetDeadline(dl)
	}

	// 3) Upgrade to SSH (this does the SSH handshake)
	conn, chans, reqs, err := ssh.NewClientConn(rawConn, addr, sshConfig)
	if err != nil {
		_ = rawConn.Close()

		return nil, fmt.Errorf("SSH handshake with %s failed: %w", addr, err)
	}

	// 4) Clear the deadline so further I/O isn’t accidentally limited
	_ = rawConn.SetDeadline(time.Time{})

	// 5) Build the high‐level SSH client
	return ssh.NewClient(conn, chans, reqs), nil
}
