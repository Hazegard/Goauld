package wireguard

import (
	"Goauld/common/log"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/waiter"
)

func (tun *netTun) acceptTCP(req *tcp.ForwarderRequest) {
	localAddress := req.ID().LocalAddress
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

	var wq waiter.Queue
	ep, tcpErr := req.CreateEndpoint(&wq)
	if tcpErr != nil {
		log.Debug().Msgf("req.CreateEndpoint() = %v", tcpErr)
		req.Complete(false)

		return
	}
	conn := gonet.NewTCPConn(&wq, ep)

	wg := sync.WaitGroup{}
	wg.Add(2)

	go tun.cpy(&wg, outbound, conn) // conn -> outbound
	go tun.cpy(&wg, conn, outbound) // outbound -> conn
	wg.Wait()
}

func (tun *netTun) cpy(wg *sync.WaitGroup, dst, src net.Conn) {
	defer wg.Done()

	//r := NewLimitReader(src, tun.limiter)
	_, err := io.Copy(dst, src)
	if err != nil {
		log.Printf("copy %v", err)
	}

	// Set a deadline for the ReadOperation so that we don't
	// wait forever for a dst that might not respond on
	// a resonable amount of time.
	dst.SetReadDeadline(time.Now().Add(tcpWaitTimeout))
}
