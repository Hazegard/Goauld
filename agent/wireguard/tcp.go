package wireguard

import (
	"Goauld/common/log"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

// Replace240With127 returns a copy of addr with the first byte changed
// from 240 to 127 if addr is an IPv4 or IPv4-mapped IPv6 address.
//
// For non-IPv4 addresses, or if the first byte is not 240, the same
// addr is returned unchanged.
func Replace240With127(addr tcpip.Address) tcpip.Address {
	v4 := addr.To4()
	if v4.Len() == 0 {
		// Not IPv4 or IPv4-mapped IPv6, return unchanged.
		return addr
	}

	b := v4.AsSlice()
	if b[0] != 240 {
		return addr
	}

	newAddr := make([]byte, 4)
	copy(newAddr, b)
	//nolint:gosec
	newAddr[0] = 127

	return tcpip.AddrFrom4Slice(newAddr)
}

// acceptTCP establish a tcp connection using the virtual interface and forward it upstream using the system network.
func (tun *netTun) acceptTCP(req *tcp.ForwarderRequest) {
	localAddress := Replace240With127(req.ID().LocalAddress)
	log.Trace().Str("RADDR", req.ID().RemoteAddress.String()).
		Uint16("RPORT", req.ID().RemotePort).
		Str("LADDR", localAddress.String()).
		Uint16("LPORT", req.ID().LocalPort).
		Msg("TCP>")

	t := fmt.Sprintf("%s:%d", localAddress, req.ID().LocalPort)
	outbound, err := net.Dial("tcp", t)
	if err != nil {
		log.Trace().Err(err).Msgf("net.Dial() = %v", err)
		req.Complete(true)

		return
	}
	defer outbound.Close()

	var wq waiter.Queue
	ep, tcpErr := req.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Debug().Msgf("req.CreateEndpoint() = %v", tcpErr)
		req.Complete(true)

		return
	}
	req.Complete(false)
	conn := gonet.NewTCPConn(&wq, ep)
	defer conn.Close()

	wg := sync.WaitGroup{}
	wg.Add(2)

	go tun.cpy(&wg, outbound, conn) // conn -> outbound
	go tun.cpy(&wg, conn, outbound) // outbound -> conn
	wg.Wait()
}

// cpy copy the net.Conns.
func (tun *netTun) cpy(wg *sync.WaitGroup, dst, src net.Conn) {
	defer wg.Done()

	// r := NewLimitReader(src, tun.limiter)
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("copy %v", err)
	}

	// Set a deadline for the ReadOperation so that we don't
	// wait forever for a dst that might not respond on
	// a resonable amount of time.
	_ = dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
