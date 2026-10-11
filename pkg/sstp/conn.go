package sstp

import (
	"bytes"
	"errors"
	"io"
	"net"
	"sync"
	"time"
)

type TCPConn struct {
	tunnel     *Tunnel
	localIP    net.IP
	remoteIP   net.IP
	localPort  uint16
	remotePort uint16

	seq uint32
	ack uint32

	mu       sync.Mutex
	readBuf  bytes.Buffer
	readCond *sync.Cond

	// Sliding window & out-of-order reassembly buffer
	outOfOrder map[uint32][]byte
	rcvWnd     uint32

	// Cumulative / Delayed ACK state
	unackedBytes int
	ackTimer     *time.Timer
	ackScheduled bool

	closed      bool
	peerClosed  bool
	closeErr    error
	readTimeout time.Duration

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
		outOfOrder: make(map[uint32][]byte),
		rcvWnd:     65535,
	}
	c.readCond = sync.NewCond(&c.mu)
	return c
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

	return c.readBuf.Read(b)
}

func (c *TCPConn) Write(b []byte) (n int, err error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return 0, io.ErrClosedPipe
	}

	total := len(b)
	offset := 0

	for offset < total {
		chunkSize := total - offset
		if chunkSize > DefaultTCPMSS {
			chunkSize = DefaultTCPMSS
		}
		chunk := b[offset : offset+chunkSize]

		pkt := BuildIPv4TCPPacket(
			c.localIP, c.remoteIP,
			c.localPort, c.remotePort,
			c.seq, c.ack,
			TCPFlagACK|TCPFlagPSH,
			chunk,
		)
		if pkt == nil {
			return offset, errors.New("failed to build sstp tcp packet")
		}

		if err := c.tunnel.sendRaw(pkt); err != nil {
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

	ackPkt := BuildIPv4TCPPacket(
		c.localIP, c.remoteIP,
		c.localPort, c.remotePort,
		c.seq, c.ack,
		TCPFlagACK,
		nil,
	)
	if ackPkt != nil {
		_ = c.tunnel.sendRaw(ackPkt)
	}
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

			if len(validData) > 0 {
				c.readBuf.Write(validData)
				c.ack += uint32(len(validData))
				c.unackedBytes += len(validData)
			}

			// Drain any buffered out-of-order packets that now line up with c.ack
			for {
				buffered, exists := c.outOfOrder[c.ack]
				if !exists {
					break
				}
				delete(c.outOfOrder, c.ack)
				c.readBuf.Write(buffered)
				c.ack += uint32(len(buffered))
				c.unackedBytes += len(buffered)
			}

			c.readCond.Broadcast()

			// Check cumulative ACK threshold: send if >= 2 * MSS, else schedule delayed ACK
			if c.unackedBytes >= 2*DefaultTCPMSS {
				c.sendAckLocked()
			} else {
				c.scheduleAckLocked()
			}
		} else {
			// Out-of-order segment (sequence hole detected)
			if startDiff < int32(c.rcvWnd) && len(c.outOfOrder) < 128 {
				c.outOfOrder[pktSeq] = append([]byte(nil), pkt.Payload...)
			}
			// Send immediate ACK with current c.ack to trigger Fast Retransmit on peer
			c.sendAckLocked()
		}
	}

	// Remote closed with FIN
	if pkt.Flags&TCPFlagFIN != 0 {
		c.peerClosed = true
		c.ack = pkt.Seq + 1
		c.readCond.Broadcast()

		// Send FIN-ACK reply immediately
		finAck := BuildIPv4TCPPacket(
			c.localIP, c.remoteIP,
			c.localPort, c.remotePort,
			c.seq, c.ack,
			TCPFlagACK|TCPFlagFIN,
			nil,
		)
		if finAck != nil {
			_ = c.tunnel.sendRaw(finAck)
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
	c.readCond.Broadcast()

	finPkt := BuildIPv4TCPPacket(
		c.localIP, c.remoteIP,
		c.localPort, c.remotePort,
		c.seq, c.ack,
		TCPFlagACK|TCPFlagFIN,
		nil,
	)
	c.mu.Unlock()

	if finPkt != nil {
		_ = c.tunnel.sendRaw(finPkt)
	}

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
