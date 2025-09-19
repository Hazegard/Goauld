package http

import (
	"Goauld/common/log"
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"www.bamsoftware.com/git/champa.git/encapsulation"
	"www.bamsoftware.com/git/champa.git/turbotunnel"
)

const (
	// pollLoop has a poll timer that automatically sends an empty polling
	// query when a certain amount of time has elapsed without a send. The
	// poll timer is initially set to initPollDelay. It increases by a
	// factor of pollDelayMultiplier every time the poll timer expires, up
	// to a maximum of maxPollDelay. The poll timer is reset to
	// initPollDelay whenever an a send occurs that is not the result of the
	// poll timer expiring.
	initPollDelay       = 1 * time.Second
	maxPollDelay        = 10 * time.Second
	pollDelayMultiplier = 2.0

	// How long we wait for a start-to-finish request–response exchange,
	// including reading the response body.
	pollTimeout = 30 * time.Second

	maxPayloadLength = 64 * 1024
)

// PollingPacketConn implements the net.PacketConn interface over an abstract
// poll function. Packets addressed to remoteAddr are passed to WriteTo are
// batched, encapsulated, and passed to the poll function. Packets addressed to
// other remote addresses are ignored. The poll function returns its own batch
// of incoming packets which are queued to be returned from a future call to
// ReadFrom.
type PollingPacketConn struct {
	remoteAddr net.Addr
	clientID   turbotunnel.ClientID
	ctx        context.Context
	client     *http.Client
	cancel     context.CancelFunc
	// QueuePacketConn is the direct receiver of ReadFrom and WriteTo calls.
	// sendLoop removes messages from the outgoing queue that were placed
	// there by WriteTo, and inserts messages into the incoming queue to be
	// returned from ReadFrom.
	*turbotunnel.QueuePacketConn
}

type PollFunc func(context.Context, *http.Client, []byte) (io.ReadCloser, error)

func NewPollingPacketConn(remoteAddr net.Addr, poll PollFunc, client *http.Client) *PollingPacketConn {
	clientID := turbotunnel.NewClientID()
	ctx, cancel := context.WithCancel(context.Background())
	c := &PollingPacketConn{
		remoteAddr:      remoteAddr,
		clientID:        clientID,
		ctx:             ctx,
		cancel:          cancel,
		client:          client,
		QueuePacketConn: turbotunnel.NewQueuePacketConn(clientID, 0),
	}
	go func() {
		err := c.pollLoop(poll)
		if err != nil {
			log.Trace().Err(err).Str("Mode", "SSHTTP").Msg("pollLoop() failed")
		}
	}()
	return c
}

// Close cancels any in-progress polls and closes the underlying
// QueuePacketConn.
func (c *PollingPacketConn) Close() error {
	c.cancel()
	return c.QueuePacketConn.Close()
}

func (c *PollingPacketConn) pollLoop(poll PollFunc) error {
	// TODO: compute this dynamically, considering URL length and encoding
	// overhead.

	pollDelay := initPollDelay
	pollTimer := time.NewTimer(pollDelay)
	for {
		var payload bytes.Buffer
		payload.Write(c.clientID[:])

		var p []byte
		unstash := c.Unstash(c.remoteAddr)
		outgoing := c.OutgoingQueue(c.remoteAddr)
		pollTimerExpired := false
		// Block, waiting for one packet or a demand to poll. Prioritize
		// taking a packet from the stash, then taking one from the
		// outgoing queue, then finally also consider polls.
		select {
		case <-c.ctx.Done():
			return nil
		case p = <-unstash:
		default:
			select {
			case <-c.ctx.Done():
				return nil
			case p = <-unstash:
			case p = <-outgoing:
			default:
				select {
				case <-c.ctx.Done():
					return nil
				case p = <-unstash:
				case p = <-outgoing:
				case <-pollTimer.C:
					pollTimerExpired = true
				}
			}
		}

		if pollTimerExpired {
			// We're polling because it's been a while since we last
			// polled. Increase the poll delay.
			pollDelay = time.Duration(float64(pollDelay) * pollDelayMultiplier)
			if pollDelay > maxPollDelay {
				pollDelay = maxPollDelay
			}
		} else {
			// We're sending an actual data packet. Reset the poll
			// delay to initial.
			if !pollTimer.Stop() {
				<-pollTimer.C
			}
			pollDelay = initPollDelay
		}
		pollTimer.Reset(pollDelay)

		// Grab as many more packets as are immediately available and
		// fit in maxPayloadLength. Always include the first packet,
		// even if it doesn't fit.
		first := true
		for len(p) > 0 && (first || payload.Len()+len(p) <= maxPayloadLength) {
			first = false

			// Encapsulate the packet into the payload.
			_, _ = encapsulation.WriteData(&payload, p)

			select {
			case p = <-outgoing:
			default:
				p = nil
			}
		}
		if len(p) > 0 {
			// We read an actual packet, but it didn't fit under the
			// limit. Stash it so that it will be first in line for
			// the next poll.
			c.Stash(p, c.remoteAddr)
		}

		go func() {
			ctx, cancel := context.WithTimeout(c.ctx, pollTimeout)
			defer cancel()
			body, err := poll(ctx, c.client, payload.Bytes())
			if err != nil {
				log.Trace().Err(err).Str("Mode", "SSHTTP").Msg("Poll error")
				// TODO: perhaps self-throttle when this happens.
				return
			}
			//nolint:errcheck
			defer body.Close()
			err = c.processIncoming(body)
			if err != nil {
				log.Trace().Err(err).Str("Mode", "SSHTTP").Msg("processIncoming error")
			}
		}()
	}
}

// processIncoming reads a packet from a poll response body and feeds it to the
// incoming queue of c.QueuePacketConn.
//
// In main, we've done SetACKNoDelay on the *kcp.UDPSession. I expect this will
// cause us, the client, to ACK incoming data immediately, which means that
// whenever we receive ACKable data, we immediately do another poll (carrying an
// ACK), which is what we want anyway while we're actively downloading.
func (c *PollingPacketConn) processIncoming(body io.Reader) error {
	// Safety limit on response body length.
	lr := io.LimitReader(body, 500*1024)
	for {
		p, err := encapsulation.ReadData(lr)
		if err != nil {
			if err == io.EOF && lr.(*io.LimitedReader).N == 0 {
				err = errors.New("response body too large")
			} else if err == io.EOF {
				err = nil
			}
			return err
		}

		c.QueueIncoming(p, c.remoteAddr)
	}
}
