package sstp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

// UDPConn implements net.PacketConn for Userland UDP over SSTP PPP link.
type UDPConn struct {
	tun           *Tunnel
	localIP       net.IP
	localPort     uint16
	inCh          chan *ParsedUDPPacket
	closeOnce     sync.Once
	closed        chan struct{}
	readDeadline  time.Time
	writeDeadline time.Time
	mu            sync.RWMutex
}

func newUDPConn(tun *Tunnel, localIP net.IP, localPort uint16) *UDPConn {
	return &UDPConn{
		tun:       tun,
		localIP:   localIP,
		localPort: localPort,
		inCh:      make(chan *ParsedUDPPacket, 256),
		closed:    make(chan struct{}),
	}
}

func (u *UDPConn) handleIncoming(pkt *ParsedUDPPacket) {
	select {
	case <-u.closed:
	case u.inCh <- pkt:
	default:
		// Queue full, drop packet per standard UDP best-effort semantics
	}
}

func (u *UDPConn) ReadFrom(p []byte) (n int, addr net.Addr, err error) {
	u.mu.RLock()
	deadline := u.readDeadline
	u.mu.RUnlock()

	var timer *time.Timer
	var timeoutCh <-chan time.Time
	if !deadline.IsZero() {
		dur := time.Until(deadline)
		if dur <= 0 {
			return 0, nil, errors.New("i/o timeout")
		}
		timer = time.NewTimer(dur)
		defer timer.Stop()
		timeoutCh = timer.C
	}

	select {
	case <-u.closed:
		return 0, nil, errors.New("use of closed network connection")
	case <-timeoutCh:
		return 0, nil, errors.New("i/o timeout")
	case pkt, ok := <-u.inCh:
		if !ok {
			return 0, nil, errors.New("use of closed network connection")
		}
		n = copy(p, pkt.Payload)
		srcAddr := &net.UDPAddr{
			IP:   pkt.SrcIP,
			Port: int(pkt.SrcPort),
		}
		return n, srcAddr, nil
	}
}

func (u *UDPConn) WriteTo(p []byte, addr net.Addr) (n int, err error) {
	select {
	case <-u.closed:
		return 0, errors.New("use of closed network connection")
	default:
	}

	u.mu.RLock()
	deadline := u.writeDeadline
	u.mu.RUnlock()
	if !deadline.IsZero() && time.Now().After(deadline) {
		return 0, errors.New("i/o timeout")
	}

	// PacketConn destinations must already be resolved by the selected egress.
	// net.ResolveUDPAddr here would silently use host DNS for custom net.Addr.
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok || udpAddr == nil {
		return 0, fmt.Errorf("UDP destination must be a resolved *net.UDPAddr")
	}

	if udpAddr.IP == nil {
		return 0, errors.New("nil destination IP")
	}

	pkt := BuildIPv4UDPPacket(u.localIP, udpAddr.IP, u.localPort, uint16(udpAddr.Port), p)
	if pkt == nil {
		return 0, errors.New("build udp packet failed")
	}

	if err := u.tun.sendRaw(pkt); err != nil {
		return 0, err
	}

	return len(p), nil
}

func (u *UDPConn) Close() error {
	u.closeOnce.Do(func() {
		close(u.closed)
		u.tun.removeUDPConn(u.localPort)
	})
	return nil
}

func (u *UDPConn) LocalAddr() net.Addr {
	return &net.UDPAddr{
		IP:   u.localIP,
		Port: int(u.localPort),
	}
}

func (u *UDPConn) SetDeadline(t time.Time) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.readDeadline = t
	u.writeDeadline = t
	return nil
}

func (u *UDPConn) SetReadDeadline(t time.Time) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.readDeadline = t
	return nil
}

func (u *UDPConn) SetWriteDeadline(t time.Time) error {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.writeDeadline = t
	return nil
}
