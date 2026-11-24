package sstp

import (
	"io"
	"net"
	"sync"
	"time"
)

const (
	maxTCPReadBufCap      = 65535
	maxOutOfOrderSegments = 64
	maxOutOfOrderBytes    = 65535
)

type ringBuffer struct {
	buf    []byte
	head   int
	length int
}

func (r *ringBuffer) Len() int {
	return r.length
}

func (r *ringBuffer) Free(limit int) int {
	if limit <= r.length {
		return 0
	}
	return limit - r.length
}

func (r *ringBuffer) Write(p []byte) int {
	if len(p) == 0 {
		return 0
	}
	avail := r.Free(maxTCPReadBufCap)
	if avail <= 0 {
		return 0
	}
	if len(p) > avail {
		p = p[:avail]
	}

	need := r.length + len(p)
	if need > len(r.buf) {
		newCap := len(r.buf)
		if newCap == 0 {
			newCap = 4096
		}
		for newCap < need && newCap < maxTCPReadBufCap {
			newCap *= 2
		}
		if newCap > maxTCPReadBufCap {
			newCap = maxTCPReadBufCap
		}
		nb := make([]byte, newCap)
		if r.length > 0 {
			tail := r.head + r.length
			if tail <= len(r.buf) {
				copy(nb, r.buf[r.head:tail])
			} else {
				first := len(r.buf) - r.head
				copy(nb[:first], r.buf[r.head:])
				copy(nb[first:], r.buf[:r.length-first])
			}
		}
		r.buf = nb
		r.head = 0
	}

	capBuf := len(r.buf)
	tail := (r.head + r.length) % capBuf
	first := min(len(p), capBuf-tail)
	copy(r.buf[tail:tail+first], p[:first])
	if first < len(p) {
		copy(r.buf[:len(p)-first], p[first:])
	}
	r.length += len(p)
	return len(p)
}

func (r *ringBuffer) Read(p []byte) (int, error) {
	if r.length == 0 || len(p) == 0 {
		return 0, nil
	}
	n := min(len(p), r.length)
	capBuf := len(r.buf)
	first := min(n, capBuf-r.head)
	copy(p[:first], r.buf[r.head:r.head+first])
	if first < n {
		copy(p[first:n], r.buf[:n-first])
	}
	r.length -= n
	if r.length == 0 {
		r.head = 0
	} else {
		r.head = (r.head + n) % capBuf
	}
	return n, nil
}

func (r *ringBuffer) Release() {
	r.buf = nil
	r.head = 0
	r.length = 0
}

type TCPConn struct {
	tunnel     *Tunnel
	localIP    net.IP
	remoteIP   net.IP
	localPort  uint16
	remotePort uint16

	seq uint32
	ack uint32

	mu       sync.Mutex
	readBuf  ringBuffer
	readCond *sync.Cond

	// Lazy-allocated sliding window & out-of-order reassembly buffer
	outOfOrder      map[uint32][]byte
	outOfOrderBytes int
	rcvWnd          uint32
	zeroWindowSent  bool

	// Cumulative / Delayed ACK state
	unackedBytes int
	ackTimer     *time.Timer
	ackScheduled bool

	closed     bool
	peerClosed bool
	closeErr   error

	readDeadline  time.Time
	writeDeadline time.Time
}

func newTCPConn(tunnel *Tunnel, localIP, remoteIP net.IP, localPort, remotePort uint16, initSeq, initAck uint32) *TCPConn {
	c := &TCPConn{
		tunnel:     tunnel,
		localIP:    localIP,
		remoteIP:   remoteIP,
		localPort:  localPort,
		remotePort: remotePort,
		seq:        initSeq,
		ack:        initAck,
		rcvWnd:     maxTCPReadBufCap,
	}
	c.readCond = sync.NewCond(&c.mu)
	return c
}

func (c *TCPConn) advertisedWindowLocked() uint16 {
	free := maxTCPReadBufCap - c.readBuf.Len()
	if free <= 0 {
		return 0
	}
	if free > 65535 {
		return 65535
	}
	return uint16(free)
}

func (c *TCPConn) Read(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for c.readBuf.Len() == 0 {
		if c.closed {
			return 0, io.ErrClosedPipe
		}
		if c.peerClosed {
			return 0, io.EOF
		}
		if !c.readDeadline.IsZero() && time.Now().After(c.readDeadline) {
			return 0, ErrTimeout
		}

		// Wait for data with potential timeout
		if !c.readDeadline.IsZero() {
			remain := time.Until(c.readDeadline)
			if remain <= 0 {
				return 0, ErrTimeout
			}
			t := time.AfterFunc(remain, func() {
				c.mu.Lock()
				c.readCond.Broadcast()
				c.mu.Unlock()
			})
			c.readCond.Wait()
			t.Stop()
		} else {
			c.readCond.Wait()
		}
	}

	n, _ = c.readBuf.Read(b)
	if c.zeroWindowSent && c.advertisedWindowLocked() >= 2*DefaultTCPMSS && !c.closed && !c.peerClosed {
		c.zeroWindowSent = false
		c.sendAckLocked()
	}
	return n, nil
}

func (c *TCPConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}
	if !c.writeDeadline.IsZero() && time.Now().After(c.writeDeadline) {
		return 0, ErrTimeout
	}

	total := len(b)
	offset := 0

	for offset < total {
		if !c.writeDeadline.IsZero() && time.Now().After(c.writeDeadline) {
			return offset, ErrTimeout
		}
		chunkSize := total - offset
		if chunkSize > DefaultTCPMSS {
			chunkSize = DefaultTCPMSS
		}
		chunk := b[offset : offset+chunkSize]

		if err := c.tunnel.sendTCP(
			c.localIP, c.remoteIP,
			c.localPort, c.remotePort,
			c.seq, c.ack,
			TCPFlagACK|TCPFlagPSH,
			c.advertisedWindowLocked(),
			chunk,
		); err != nil {
			return offset, err
		}

		c.seq += uint32(chunkSize)
		offset += chunkSize
	}

	return total, nil
}

func (c *TCPConn) sendAckLocked() {
	if c.ackScheduled && c.ackTimer != nil {
		c.ackTimer.Stop()
		c.ackScheduled = false
	}
	c.unackedBytes = 0

	wnd := c.advertisedWindowLocked()
	c.zeroWindowSent = (wnd == 0)
	_ = c.tunnel.sendTCP(
		c.localIP, c.remoteIP,
		c.localPort, c.remotePort,
		c.seq, c.ack,
		TCPFlagACK,
		wnd,
		nil,
	)
}

func (c *TCPConn) scheduleAckLocked() {
	if c.ackScheduled {
		return
	}
	c.ackScheduled = true
	if c.ackTimer == nil {
		c.ackTimer = time.AfterFunc(10*time.Millisecond, func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			if c.ackScheduled && !c.closed {
				c.sendAckLocked()
			}
		})
	} else {
		c.ackTimer.Reset(10 * time.Millisecond)
	}
}

func (c *TCPConn) handleIncoming(pkt *ParsedTCPPacket) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return
	}

	payloadLen := len(pkt.Payload)
	if payloadLen > 0 {
		pktSeq := pkt.Seq
		pktEnd := pktSeq + uint32(payloadLen)

		// 1. Check if packet is completely duplicate / already acked
		diff := int32(pktEnd - c.ack)
		if diff <= 0 {
			// Duplicate/old packet; send immediate ACK to keep sender synchronized
			c.sendAckLocked()
			return
		}

		startDiff := int32(pktSeq - c.ack)
		if startDiff <= 0 {
			// In-order segment (or partially overlapping)
			var validData []byte
			if startDiff < 0 {
				overlap := int(-startDiff)
				if overlap < payloadLen {
					validData = pkt.Payload[overlap:]
				}
			} else {
				validData = pkt.Payload
			}

			free := c.readBuf.Free(maxTCPReadBufCap)
			if free <= 0 {
				// Receive buffer full: advertise zero window and do not advance ACK.
				c.sendAckLocked()
				return
			}
			if len(validData) > free {
				validData = validData[:free]
			}

			if len(validData) > 0 {
				written := c.readBuf.Write(validData)
				c.ack += uint32(written)
				c.unackedBytes += written
			}

			// Drain any buffered out-of-order packets that now line up and fit in free space
			for len(c.outOfOrder) > 0 {
				buffered, exists := c.outOfOrder[c.ack]
				if !exists {
					break
				}
				remFree := c.readBuf.Free(maxTCPReadBufCap)
				if len(buffered) > remFree {
					break
				}
				delete(c.outOfOrder, c.ack)
				c.outOfOrderBytes -= len(buffered)
				if c.outOfOrderBytes < 0 {
					c.outOfOrderBytes = 0
				}
				c.readBuf.Write(buffered)
				c.ack += uint32(len(buffered))
				c.unackedBytes += len(buffered)
			}

			c.readCond.Broadcast()

			// Check cumulative ACK threshold or zero-window boundary
			if c.advertisedWindowLocked() == 0 || c.unackedBytes >= 2*DefaultTCPMSS {
				c.sendAckLocked()
			} else {
				c.scheduleAckLocked()
			}
		} else {
			// Out-of-order segment (sequence hole detected)
			if startDiff < int32(c.rcvWnd) && len(c.outOfOrder) < maxOutOfOrderSegments && c.outOfOrderBytes+payloadLen <= maxOutOfOrderBytes {
				if c.outOfOrder == nil {
					c.outOfOrder = make(map[uint32][]byte)
				}
				if _, dup := c.outOfOrder[pktSeq]; !dup {
					cp := make([]byte, payloadLen)
					copy(cp, pkt.Payload)
					c.outOfOrder[pktSeq] = cp
					c.outOfOrderBytes += payloadLen
				}
			}
			// Send immediate ACK with current c.ack to trigger Fast Retransmit on peer
			c.sendAckLocked()
		}
	}

	// Remote closed with FIN
	if pkt.Flags&TCPFlagFIN != 0 {
		finSeq := pkt.Seq + uint32(payloadLen)
		if int32(finSeq-c.ack) <= 0 {
			c.peerClosed = true
			c.ack = finSeq + 1
			c.readCond.Broadcast()

			// Send FIN-ACK reply immediately
			_ = c.tunnel.sendTCP(
				c.localIP, c.remoteIP,
				c.localPort, c.remotePort,
				c.seq, c.ack,
				TCPFlagACK|TCPFlagFIN,
				c.advertisedWindowLocked(),
				nil,
			)
		}
	}
}

func (c *TCPConn) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	if c.ackScheduled && c.ackTimer != nil {
		c.ackTimer.Stop()
		c.ackScheduled = false
	}
	c.readBuf.Release()
	c.outOfOrder = nil
	c.outOfOrderBytes = 0
	c.readCond.Broadcast()

	seq, ack := c.seq, c.ack
	c.mu.Unlock()

	_ = c.tunnel.sendTCP(
		c.localIP, c.remoteIP,
		c.localPort, c.remotePort,
		seq, ack,
		TCPFlagACK|TCPFlagFIN,
		0,
		nil,
	)

	c.tunnel.deregisterConn(c.localPort, c.remotePort)
	return nil
}

func (c *TCPConn) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP:   c.localIP,
		Port: int(c.localPort),
	}
}

func (c *TCPConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP:   c.remoteIP,
		Port: int(c.remotePort),
	}
}

func (c *TCPConn) SetDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	c.writeDeadline = t
	c.readCond.Broadcast()
	return nil
}

func (c *TCPConn) SetReadDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.readDeadline = t
	c.readCond.Broadcast()
	return nil
}

func (c *TCPConn) SetWriteDeadline(t time.Time) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.writeDeadline = t
	return nil
}
