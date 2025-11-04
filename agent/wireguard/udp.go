package wireguard

import (
	"Goauld/common/log"
	"context"
	"errors"
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
	localAddress := Replace240With127(sess.LocalAddress)
	log.Trace().Str("LocalAddr", localAddress.String()).Str("RemoteAddr", sess.RemoteAddress.String()).Msg("Received UDP packet")

	var wq waiter.Queue

	ep, udpErr := req.CreateEndpoint(&wq)
	if udpErr != nil {
		log.Debug().Err(errors.New(udpErr.String())).Msg("Failed to create endpoint")

		return false
	}
	client := gonet.NewUDPConn(&wq, ep)

	clientAddr := &net.UDPAddr{IP: net.IP(sess.RemoteAddress.AsSlice()), Port: int(sess.RemotePort)}
	remoteAddr := &net.UDPAddr{IP: net.IP(localAddress.AsSlice()), Port: int(sess.LocalPort)}
	proxyAddr := &net.UDPAddr{IP: net.ParseIP("0.0.0.0"), Port: int(sess.RemotePort)}

	proxyConn, err := net.ListenUDP("udp", proxyAddr)
	if err != nil {
		log.Debug().Err(err).Str("ProxyAddr", proxyAddr.String()).Msg("Failed to create UDP listener")
		proxyAddr.Port = 0

		proxyConn, err = net.ListenUDP("udp", proxyAddr)
		if err != nil {
			log.Debug().Err(err).Str("ProxyAddr", proxyAddr.String()).Msg("Failed to create UDP listener")

			return false
		}
	}
	ctx, cancel := context.WithCancel(context.Background())

	go tun.proxy(ctx, cancel, client, clientAddr, proxyConn)
	go tun.proxy(ctx, cancel, proxyConn, remoteAddr, client)

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
					log.Debug().Err(err).Str("SrcAddr", srcAddr.String()).Msg("Failed to read packet")
				}

				return
			}
			if n > 0 {
				err := tun.limiter.WaitN(ctx, n)
				if err != nil {
					log.Debug().Err(err).Str("SrcAddr", srcAddr.String()).Msg("Shaper error")

					return
				}
			}

			_, err = dst.WriteTo(buf[:n], dstAddr)
			if err != nil {
				if ctx.Err() == nil {
					log.Debug().Err(err).Str("DstAddr", dstAddr.String()).Msg("Failed to write packet")
				}

				return
			}
			dst.SetReadDeadline(time.Now().Add(idleTimeout))
		}
	}
}
