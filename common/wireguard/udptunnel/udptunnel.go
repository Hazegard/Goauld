package udptunnel

import (
	"Goauld/common/log"
	"encoding/gob"
	"errors"
	"io"
	"net"
	"os"
	"sync"
	"time"
)

func HandleUDP(rwc io.ReadWriteCloser, hostPort string) error {
	conns := &udpConns{
		m: map[string]*udpConn{},
	}
	defer conns.closeAll()
	h := &udpHandler{
		hostPort: hostPort,
		UdpChannel: &UdpChannel{
			R: gob.NewDecoder(rwc),
			W: gob.NewEncoder(rwc),
			C: rwc,
		},
		udpConns: conns,
		maxMTU:   9012, // settings.EnvInt("UDP_MAX_SIZE", 9012),
	}
	log.Trace().Int("MTU", h.maxMTU).Msgf("Starting UDP connection")
	for {
		p := UdpPacket{}
		if err := h.handleWrite(&p); err != nil {
			return err
		}
	}
}

type udpHandler struct {
	*UdpChannel
	*udpConns

	hostPort string
	maxMTU   int
}

func (h *udpHandler) handleWrite(p *UdpPacket) error {
	if err := h.R.Decode(&p); err != nil {
		return err
	}
	// dial now, we know we must write
	conn, exists, err := h.udpConns.dial(p.Src, h.hostPort)
	if err != nil {
		return err
	}
	// however, we dont know if we must read...
	// spawn up to <max-conns> go-routines to wait
	// for a reply.
	// TODO configurable
	// TODO++ dont use go-routines, switch to pollable
	//  array of listeners where all listeners are
	//  sweeped periodically, removing the idle ones
	const maxConns = 100
	if !exists {
		if h.udpConns.len() <= maxConns {
			go h.handleRead(p, conn)
		} else {
			log.Trace().Int("Max", maxConns).Msgf("UDP connection limit reached")
		}
	}
	_, err = conn.Write(p.Payload)
	if err != nil {
		return err
	}

	return nil
}

func (h *udpHandler) handleRead(p *UdpPacket, conn *udpConn) {
	// ensure connection is cleaned up
	defer h.udpConns.remove(conn.id)
	buff := make([]byte, h.maxMTU)
	for {
		// response must arrive within 15 seconds
		deadline := 15 * time.Second // settings.EnvDuration("UDP_DEADLINE", 15*time.Second)
		conn.SetReadDeadline(time.Now().Add(deadline))
		// read response
		n, err := conn.Read(buff)
		if err != nil {
			if !os.IsTimeout(err) && !errors.Is(err, io.EOF) {
				log.Trace().Err(err).Msg("UDP connection read error")
			}

			break
		}
		b := buff[:n]
		// encode back over ssh connection
		err = h.UdpChannel.Encode(p.Src, b)
		if err != nil {
			log.Trace().Err(err).Msg("UDP channel encode error")

			return
		}
	}
}

type udpConns struct {
	sync.Mutex

	m map[string]*udpConn
}

func (cs *udpConns) dial(id, addr string) (*udpConn, bool, error) {
	cs.Lock()
	defer cs.Unlock()
	conn, ok := cs.m[id]
	if !ok {
		c, err := net.Dial("udp", addr)
		if err != nil {
			return nil, false, err
		}
		conn = &udpConn{
			id:   id,
			Conn: c, // cnet.MeterConn(cs.Logger.Fork(addr), C),
		}
		cs.m[id] = conn
	}

	return conn, ok, nil
}

func (cs *udpConns) len() int {
	cs.Lock()
	l := len(cs.m)
	cs.Unlock()

	return l
}

func (cs *udpConns) remove(id string) {
	cs.Lock()
	delete(cs.m, id)
	cs.Unlock()
}

func (cs *udpConns) closeAll() {
	cs.Lock()
	for id, conn := range cs.m {
		conn.Close()
		delete(cs.m, id)
	}
	cs.Unlock()
}

type udpConn struct {
	net.Conn

	id string
}
