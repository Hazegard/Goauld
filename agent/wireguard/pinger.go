package wireguard

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

func Ping(address string) (bool, uint8) {
	// Always return true for localhost
	psudoLoopback := net.IPNet{
		IP:   net.IPv4(240, 0, 0, 0),
		Mask: []byte{0xf0, 0x00, 0x00, 0x00},
	}

	if psudoLoopback.Contains(net.ParseIP(address)) {
		return true, 69
	}

	/*success, ttl := pingo(address)
	if success {
		return true, ttl
	}*/
	addr, err := netip.ParseAddr(address)
	if err != nil {
		return false, 0
	}

	return pingc(addr)
}

func pingo(target string) (bool, uint8) {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		return false, 0
	}
	pinger.Count = 1
	pinger.Timeout = 4 * time.Second
	if runtime.GOOS == "windows" {
		pinger.SetPrivileged(true)
	}
	err = pinger.Run()
	if err != nil {
		return false, 0
	}

	return pinger.PacketsRecv != 0, uint8(pinger.TTL)
}

// https://github.com/tailscale/tailscale/blob/main/wgengine/netstack/netstack_userping.go#L27
// sendOutboundUserPing sends a non-privileged ICMP (or ICMPv6) ping to dstIP with the given timeout.
func pingc(dstIP netip.Addr) (bool, uint8) {
	var err error
	var out []byte
	switch runtime.GOOS {
	case "windows":
		out, err = exec.Command("ping", "-n", "1", "-w", "3000", dstIP.String()).CombinedOutput()
		if err == nil && !windowsPingOutputIsSuccess(dstIP, out) {
			// TODO(bradfitz,nickkhyl): return the actual ICMP error we heard back to the caller?
			// For now we just drop it.
			err = errors.New("unsuccessful ICMP reply received")
		}
	case "freebsd":
		// Note: 2000 ms is actually 1 second + 2,000
		// milliseconds extra for 3 seconds total.
		// See https://github.com/tailscale/tailscale/pull/3753 for details.
		ping := "ping"
		if dstIP.Is6() {
			ping = "ping6"
		}
		out, err = exec.Command(ping, "-c", "1", "-W", "2000", dstIP.String()).CombinedOutput()
	case "openbsd":
		ping := "ping"
		if dstIP.Is6() {
			ping = "ping6"
		}
		out, err = exec.Command(ping, "-c", "1", "-w", "3", dstIP.String()).CombinedOutput()
	case "android":
		ping := "/system/bin/ping"
		if dstIP.Is6() {
			ping = "/system/bin/ping6"
		}
		out, err = exec.Command(ping, "-c", "1", "-w", "3", dstIP.String()).CombinedOutput()
	default:
		ping := "ping"

		out, err = exec.Command(ping, "-c", "1", "-W", "3", dstIP.String()).CombinedOutput()
	}
	if err != nil {
		return false, 0
	}
	ttl := extractTTL(string(out))
	if ttl == 0 {
		ttl = 64
	}

	return true, uint8(ttl)
}

func extractTTL(input string) int {
	output := strings.Split(strings.ToLower(input), "\n")

	for _, line := range output {
		idx := strings.Index(line, "ttl=")
		if idx == -1 {
			continue
		}

		start := idx + len("ttl=")
		end := start
		for end < len(line) && line[end] >= '0' && line[end] <= '9' {
			end++
		}

		if start == end {
			continue
		}

		ttl, err := strconv.Atoi(line[start:end])
		if err != nil {
			continue
		}

		return ttl
	}

	return 0
}

// https://github.com/tailscale/tailscale/blob/main/wgengine/netstack/netstack.go#L2057
// windowsPingOutputIsSuccess reports whether the ping.exe output b contains a
// success ping response for ip.
//
// See https://github.com/tailscale/tailscale/issues/13654
//
// TODO(bradfitz,nickkhyl): delete this and use the proper Windows APIs.
func windowsPingOutputIsSuccess(ip netip.Addr, b []byte) bool {
	// Look for a line that contains " <ip>: " and then three equal signs.
	// As a special case, the 2nd equal sign may be a '<' character
	// for sub-millisecond pings.
	// This heuristic seems to match the ping.exe output in any language.
	sub := fmt.Appendf(nil, " %s: ", ip)

	eqSigns := func(bb []byte) (n int) {
		for _, b := range bb {
			if b == '=' || (b == '<' && n == 1) {
				n++
			}
		}

		return
	}

	for len(b) > 0 {
		var line []byte
		line, b, _ = bytes.Cut(b, []byte("\n"))
		if _, rest, ok := bytes.Cut(line, sub); ok && eqSigns(rest) == 3 {
			return true
		}
	}

	return false
}
