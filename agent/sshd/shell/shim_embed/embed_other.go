//go:build !windows

package shim_embed

func DropShimSSHD() (string, error) {
	return "", nil
}
