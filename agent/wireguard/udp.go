package wireguard

import (
	"Goauld/common/log"
	"context"
	"fmt"
	"net"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
	"gvisor.dev/gvisor/pkg/waiter"
)

const (
	idleTimeout = 10 * time.Second
)

func (tun *netTun) acceptUDP(req *udp.ForwarderRequest) bool {
	sess := req.ID()
	log.Println("acceptUDP>", sess.LocalAddress, sess.RemoteAddress)

	var wq waiter.Queue

	ep, udpErr := req.CreateEndpoint(&wq)
	fmt.Println("WAIT DONE")
	if udpErr != nil {
		log.Printf("udpErr %v", udpErr)

		return false
	}
	client := gonet.NewUDPConn(&wq, ep)
	fmt.Println(" gonet.NewUDPConn(&wq, ep)")

	clientAddr := &net.UDPAddr{IP: net.IP(sess.RemoteAddress.AsSlice()), Port: int(sess.RemotePort)}
	remoteAddr := &net.UDPAddr{IP: net.IP(sess.LocalAddress.AsSlice()), Port: int(sess.LocalPort)}
	proxyAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: int(sess.RemotePort)}

	/*if remoteAddr.Port == 53 && tun.isLocal(sess.LocalAddress) {
		remoteAddr.Port = 53
		remoteAddr.IP = net.ParseIP("127.0.0.1")
	}*/

	proxyConn, err := net.ListenUDP("udp", proxyAddr)
	fmt.Println("ListenUDP")
	if err != nil {
		log.Printf("Failed to bind local port %d, trying one more time with random port", proxyAddr)
		proxyAddr.Port = 0

		proxyConn, err = net.ListenUDP("udp", proxyAddr)
		if err != nil {
			log.Printf("Failed to bind local random port %s", proxyAddr)

			return false
		}
	}
	ctx, cancel := context.WithCancel(context.Background())

	go tun.proxy(ctx, cancel, client, clientAddr, proxyConn)
	go tun.proxy(ctx, cancel, proxyConn, remoteAddr, client)
	fmt.Println("LETSGO")

	return false
}

func (tun *netTun) proxy(ctx context.Context, cancel context.CancelFunc, dst net.PacketConn, dstAddr net.Addr, src net.PacketConn) {
	defer cancel()
	buf := make([]byte, tun.mtu)

	for {
		select {
		case <-ctx.Done():
			return
		default:
			src.SetReadDeadline(time.Now().Add(idleTimeout))
			n, srcAddr, err := src.ReadFrom(buf)
			if e, ok := err.(net.Error); ok && e.Timeout() {
				return
			} else if err != nil {
				if ctx.Err() == nil {
					log.Printf("Failed to read packed from %s", srcAddr)
				}

				return
			}
			fmt.Println("READ", src, n)
			if n > 0 {
				err := tun.limiter.WaitN(ctx, n)
				if err != nil {
					log.Printf("Shaper error: %v", err)

					return
				}
			}

			_, err = dst.WriteTo(buf[:n], dstAddr)
			if err != nil {
				if ctx.Err() == nil {
					log.Printf("Failed to write packed to %s", dstAddr)
				}

				return
			}
			dst.SetReadDeadline(time.Now().Add(idleTimeout))
		}
	}
}
