//go:build !windows

package shim_embed

func DropShimSSHD() (string, func() error, error) {
	return "", func() error { return nil }, nil
}
