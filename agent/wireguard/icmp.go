package wireguard

import (
	"Goauld/common/log"

	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"

	"gvisor.dev/gvisor/pkg/buffer"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/header/parse"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
)

// ICMPHandler replay the ICMP packet using ping on the host network.
func ICMPHandler(s *stack.Stack) func(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
	return func(id stack.TransportEndpointID, pkt *stack.PacketBuffer) bool {
		localAddress := Replace240With127(id.LocalAddress)
		log.Debug().Str("SRC", localAddress.String()).Str("DST", id.RemoteAddress.String()).Msg("Receive a icmp package")

		// // remote - peer's tunnnel interface address
		if localAddress.String() == "100.64.0.1" {
			log.Debug().Str("SRC", id.RemoteAddress.String()).Str("DST", localAddress.String()).Msg("[ICMP] handle localy")
			repVV := handleICMP(pkt, 69)
			s.WritePacketToRemote(1, "", ipv4.ProtocolNumber, repVV.ToBuffer())

			return true
		}

		success, ttl := Ping(localAddress.String())
		if success {
			repVV := handleICMP(pkt, ttl)
			s.WritePacketToRemote(1, "", ipv4.ProtocolNumber, repVV.ToBuffer())

			return true
		}

		return false
	}
}

// handleICMP returns the ICMP packet according to the ping response and the incoming ICMP request.
func handleICMP(pkt *stack.PacketBuffer, ttl uint8) *stack.PacketBuffer {
	replyData := stack.PayloadSince(pkt.TransportHeader())
	iph := header.IPv4(pkt.NetworkHeader().Slice())

	// Build IPv4 header for the reply.
	replyHdr := make(header.IPv4, header.IPv4MinimumSize)
	replyHdr.Encode(&header.IPv4Fields{
		TTL:      ttl,
		SrcAddr:  iph.DestinationAddress(),
		DstAddr:  iph.SourceAddress(),
		Protocol: uint8(header.ICMPv4ProtocolNumber),
	})
	//nolint:gosec
	replyHdr.SetTotalLength(uint16(len(replyHdr) + len(replyData.AsSlice())))
	replyHdr.SetChecksum(^replyHdr.CalculateChecksum())

	// Fix ICMP header.
	replyICMP := header.ICMPv4(replyData.AsSlice())
	replyICMP.SetType(header.ICMPv4EchoReply)
	// replyICMP.SetChecksum(0)
	replyICMP.SetChecksum(header.ICMPv4Checksum(replyData.AsSlice(), 0))

	// Build payload buffer.
	payload := buffer.MakeWithData(replyHdr)
	_ = payload.Append(buffer.NewViewWithData(replyICMP))

	// Create new packet buffer for reply.
	replyPkt := stack.NewPacketBuffer(stack.PacketBufferOptions{
		ReserveHeaderBytes: header.IPv4MaximumHeaderSize,
		Payload:            payload,
	})

	// Parse headers so stack knows what's inside.
	if ok := parse.IPv4(replyPkt); !ok {
		panic("expected to parse IPv4 header we just created")
	}
	if ok := parse.ICMPv4(replyPkt); !ok {
		panic("expected to parse ICMPv4 header we just created")
	}
	log.Debug().
		Str("src", replyHdr.SourceAddress().String()).
		Str("dst", replyHdr.DestinationAddress().String()).
		Msg("[ICMP] sending reply")

	return replyPkt
}
