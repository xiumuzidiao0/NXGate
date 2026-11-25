package sstp

import (
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

// UDPConn wraps gVisor gonet.UDPConn while preserving resolved-IP-only egress invariants.
type UDPConn struct {
	conn      *gonet.UDPConn
	closeOnce sync.Once
	closed    chan struct{}
}

func newUDPConn(conn *gonet.UDPConn) *UDPConn {
	return &UDPConn{
		conn:   conn,
		closed: make(chan struct{}),
	}
}

func (u *UDPConn) ReadFrom(p []byte) (int, net.Addr, error) {
	select {
	case <-u.closed:
		return 0, nil, errors.New("use of closed network connection")
	default:
	}
	if u.conn == nil {
		return 0, nil, errors.New("use of closed network connection")
	}
	return u.conn.ReadFrom(p)
}

func (u *UDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	select {
	case <-u.closed:
		return 0, errors.New("use of closed network connection")
	default:
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
	ip4 := udpAddr.IP.To4()
	if ip4 == nil {
		return 0, fmt.Errorf("SSTP UDP supports IPv4 targets only")
	}
	if u.conn == nil {
		return 0, errors.New("use of closed network connection")
	}

	return u.conn.WriteTo(p, &net.UDPAddr{IP: ip4, Port: udpAddr.Port})
}

func (u *UDPConn) Close() error {
	var err error
	u.closeOnce.Do(func() {
		if u.closed != nil {
			close(u.closed)
		}
		if u.conn != nil {
			err = u.conn.Close()
		}
	})
	return err
}

func (u *UDPConn) LocalAddr() net.Addr {
	if u.conn != nil {
		return u.conn.LocalAddr()
	}
	return nil
}

func (u *UDPConn) SetDeadline(t time.Time) error {
	if u.conn == nil {
		return errors.New("use of closed network connection")
	}
	return u.conn.SetDeadline(t)
}

func (u *UDPConn) SetReadDeadline(t time.Time) error {
	if u.conn == nil {
		return errors.New("use of closed network connection")
	}
	return u.conn.SetReadDeadline(t)
}

func (u *UDPConn) SetWriteDeadline(t time.Time) error {
	if u.conn == nil {
		return errors.New("use of closed network connection")
	}
	return u.conn.SetWriteDeadline(t)
}
