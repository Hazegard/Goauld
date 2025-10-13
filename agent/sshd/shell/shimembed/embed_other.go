//go:build !windows

// Package shimembed embed the windows sshd shim
package shimembed

// DropShimSSHD fake.
func DropShimSSHD() (string, func() error, error) {
	return "", func() error { return nil }, nil
}
