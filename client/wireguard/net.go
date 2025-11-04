package wireguard

import (
	"math/rand"
	"net"
)

// RandomCarrierGradeNATIP returns a random IPv4 address in the 100.64.0.0/10 range.
func RandomCarrierGradeNATIP() net.IP {
	// CIDR: 100.64.0.0/10 covers 100.64.0.0 - 100.127.255.255
	base := uint32(100)<<24 | uint32(64)<<16 | 0<<8 | 0 // 100.64.0.0
	maskSize := 1 << (32 - 10)                          // 2^(32-10) = 4,194,304 addresses

	// Pick a random offset within the range
	//nolint:gosec
	offset := rand.Uint32() % uint32(maskSize)

	// Compute final IP
	ipInt := base + offset
	ip := net.IPv4(
		byte(ipInt>>24),
		byte(ipInt>>16),
		byte(ipInt>>8),
		byte(ipInt),
	)

	return ip
}
