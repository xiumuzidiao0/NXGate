//go:build !linux

package proxy

import (
	"context"
	"net"
	"time"
)

func dialUpstream(targetAddr string, devName string, timeout time.Duration) (net.Conn, error) {
	d := net.Dialer{
		Timeout:   timeout,
		KeepAlive: 30 * time.Second,
	}
	return d.Dial("tcp", targetAddr)
}

func createBoundUDPSocket(devName string) (*net.UDPConn, error) {
	return net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
}

func resolveUDPAddrThroughTunnel(ctx context.Context, host string, port int, devName string) (*net.UDPAddr, error) {
	ip := net.ParseIP(host)
	if ip != nil {
		return &net.UDPAddr{IP: ip, Port: port}, nil
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip4", host)
	if err != nil || len(ips) == 0 {
		return nil, err
	}
	return &net.UDPAddr{IP: ips[0], Port: port}, nil
}
