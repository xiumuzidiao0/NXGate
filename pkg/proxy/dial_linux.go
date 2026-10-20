//go:build linux

package proxy

import (
	"context"
	"fmt"
	"net"
	"strings"
	"syscall"
	"time"
)

func getInterfaceIPv4(devName string) net.IP {
	if devName == "" {
		return nil
	}
	iface, err := net.InterfaceByName(devName)
	if err != nil {
		return nil
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return nil
	}
	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ip4 := ipnet.IP.To4(); ip4 != nil {
				return ip4
			}
		}
	}
	return nil
}

func dialUpstream(targetAddr string, devName string, timeout time.Duration) (net.Conn, error) {
	if devName == "" {
		return nil, fmt.Errorf("no authorized tunnel available")
	}
	d := net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
		Control: func(network, address string, c syscall.RawConn) error {
			var operr error
			fn := func(fd uintptr) {
				if devName != "" {
					// Dynamically bind to the specific tunnel device (tun0, tun1, tun2...)
					if err := syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, devName); err != nil {
						operr = err
					}
				}
			}
			if err := c.Control(fn); err != nil {
				return err
			}
			return operr
		},
	}

	if devName != "" {
		ip4 := getInterfaceIPv4(devName)
		if ip4 != nil {
			d.LocalAddr = &net.TCPAddr{IP: ip4}
		}

		d.Resolver = newTunnelResolver(devName, ip4)

	}

	conn, err := d.Dial("tcp", targetAddr)
	if err != nil {
		return nil, err
	}
	if tc, ok := conn.(*net.TCPConn); ok {
		_ = tc.SetNoDelay(true)
	}
	return conn, nil
}

func createBoundUDPSocket(devName string) (*net.UDPConn, error) {
	if devName == "" {
		return nil, fmt.Errorf("no authorized UDP tunnel available")
	}
	lc := net.ListenConfig{}
	if devName != "" {
		lc.Control = func(network, address string, c syscall.RawConn) error {
			var operr error
			fn := func(fd uintptr) {
				if err := syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, devName); err != nil {
					operr = err
				}
			}
			if err := c.Control(fn); err != nil {
				return err
			}
			return operr
		}
	}

	var localIP net.IP
	if devName != "" {
		localIP = getInterfaceIPv4(devName)
	}

	conn, err := lc.ListenPacket(context.Background(), "udp4", (&net.UDPAddr{IP: localIP, Port: 0}).String())
	if err != nil {
		conn, err = lc.ListenPacket(context.Background(), "udp4", ":0")
		if err != nil {
			return nil, err
		}
	}
	return conn.(*net.UDPConn), nil
}

func resolveUDPAddrThroughTunnel(ctx context.Context, host string, port int, devName string) (*net.UDPAddr, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return &net.UDPAddr{IP: ip, Port: port}, nil
	}

	if devName == "" {
		return nil, fmt.Errorf("no tunnel DNS resolver available")
	}

	resolver := newTunnelResolver(devName, getInterfaceIPv4(devName))

	ips, err := resolver.LookupIP(ctx, "ip4", host)
	if err != nil {
		return nil, err
	}
	if len(ips) == 0 {
		return nil, fmt.Errorf("no IPv4 address for %s", host)
	}
	return &net.UDPAddr{IP: ips[0], Port: port}, nil
}

// Construct once per operation with a fresh interface address, not a process-global
// cache keyed by reusable tunN names. This avoids stale routing after reconnect.
// net.Resolver coalesces lookups in this operation. Never consult host DNS.
func newTunnelResolver(devName string, localIP net.IP) *net.Resolver {
	return &net.Resolver{PreferGo: true, Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
		if devName == "" {
			return nil, fmt.Errorf("no tunnel DNS resolver available")
		}
		dial := func(proto, target string) (net.Conn, error) {
			d := net.Dialer{Timeout: 4 * time.Second, Control: func(_, _ string, c syscall.RawConn) error {
				var sockErr error
				err := c.Control(func(fd uintptr) {
					sockErr = syscall.SetsockoptString(int(fd), syscall.SOL_SOCKET, syscall.SO_BINDTODEVICE, devName)
				})
				if err != nil {
					return err
				}
				return sockErr
			}}
			if localIP != nil {
				if proto == "tcp" {
					d.LocalAddr = &net.TCPAddr{IP: localIP}
				} else {
					d.LocalAddr = &net.UDPAddr{IP: localIP}
				}
			}
			return d.DialContext(ctx, proto, target)
		}
		proto := "udp"
		if strings.HasPrefix(network, "tcp") {
			proto = "tcp"
		}
		conn, err := dial(proto, "8.8.8.8:53")
		if err != nil && ctx.Err() == nil {
			conn, err = dial(proto, "1.1.1.1:53")
		}
		if err != nil && proto == "udp" && ctx.Err() == nil {
			conn, err = dial("tcp", "1.1.1.1:53")
		}
		return conn, err
	}}
}
