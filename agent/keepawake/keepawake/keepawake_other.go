//go:build !windows

package keepawake

// For non-Windows platforms, these are no-ops.
//
//nolint:unused
func setWindowsKeepAlive() error {
	return nil
}

//nolint:unused
func stopWindowsKeepAlive() error {
	return nil
}
