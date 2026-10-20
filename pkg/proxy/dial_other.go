//go:build !linux

package proxy

import (
	"context"
	"fmt"
	"net"
	"time"
)

func dialUpstream(targetAddr string, devName string, timeout time.Duration) (net.Conn, error) {
	return nil, fmt.Errorf("device-bound TCP requires Linux; use an explicit userland dialer")
}

func createBoundUDPSocket(devName string) (*net.UDPConn, error) {
	return nil, fmt.Errorf("device-bound UDP requires Linux; use an explicit userland packet connection")
}

func resolveUDPAddrThroughTunnel(ctx context.Context, host string, port int, devName string) (*net.UDPAddr, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return &net.UDPAddr{IP: ip, Port: port}, nil
	}
	return nil, fmt.Errorf("device-bound DNS requires Linux")
}
