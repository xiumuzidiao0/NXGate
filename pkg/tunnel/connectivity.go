package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

// UserlandConnectivityTargets are dialled through a userland tunnel using literal IPs,
// so connectivity checks exercise IP forwarding only and never depend on DNS.
var UserlandConnectivityTargets = []string{"1.1.1.1:443", "8.8.8.8:443", "9.9.9.9:443"}

// CheckDialerConnectivityContext tests whether a userland dialer can establish an outbound TCP connection.
func CheckDialerConnectivityContext(ctx context.Context, dialer func(context.Context, string, string) (net.Conn, error), timeout time.Duration) bool {
	ok, _ := probeDialerConnectivity(ctx, dialer, timeout)
	return ok
}

func probeDialerConnectivity(ctx context.Context, dialer func(context.Context, string, string) (net.Conn, error), timeout time.Duration) (bool, error) {
	ok, _, err := MeasureDialerConnectivityContext(ctx, dialer, timeout)
	return ok, err
}

// MeasureDialerConnectivityContext verifies whether a userland dialer can forward TCP traffic
// and returns the measured connect RTT in milliseconds on the first reachable target.
func MeasureDialerConnectivityContext(ctx context.Context, dialer func(context.Context, string, string) (net.Conn, error), timeout time.Duration) (bool, int, error) {
	if ctx.Err() != nil {
		return false, 0, ctx.Err()
	}
	if dialer == nil {
		return false, 0, nil
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	for _, target := range UserlandConnectivityTargets {
		if ctx.Err() != nil {
			return false, 0, ctx.Err()
		}
		dialCtx, cancel := context.WithTimeout(ctx, timeout)
		start := time.Now()
		conn, err := dialer(dialCtx, "tcp", target)
		cancel()
		if err == nil {
			_ = conn.Close()
			return true, max(1, int(time.Since(start).Milliseconds())), nil
		}
		if ctx.Err() != nil {
			return false, 0, ctx.Err()
		}
	}
	return false, 0, nil
}

// CheckTunnelConnectivity tests whether a specific network interface has functional egress to the Internet.
func CheckTunnelConnectivity(devName string, timeout time.Duration) bool {
	return CheckTunnelConnectivityContext(context.Background(), devName, timeout)
}

func CheckTunnelConnectivityContext(ctx context.Context, devName string, timeout time.Duration) bool {
	ok, _ := probeTunnelConnectivity(ctx, devName, timeout)
	return ok
}

// Unknown (deadline/cancellation) is different from repeated explicit failures.
func probeTunnelConnectivity(ctx context.Context, devName string, timeout time.Duration) (bool, error) {
	ok, _, err := MeasureTunnelConnectivityContext(ctx, devName, timeout)
	return ok, err
}

// MeasureTunnelConnectivityContext verifies whether a kernel tunnel device can forward TCP traffic
// and returns the measured connect RTT in milliseconds on the first reachable target.
func MeasureTunnelConnectivityContext(ctx context.Context, devName string, timeout time.Duration) (bool, int, error) {
	if ctx.Err() != nil {
		return false, 0, ctx.Err()
	}
	if devName == "" {
		return false, 0, fmt.Errorf("no tunnel device")
	}
	unknown := false
	if timeout <= 0 {
		timeout = 3 * time.Second
	}
	// Try port 80 and port 53 (TCP)
	targets := []string{"1.1.1.1:80", "1.0.0.1:80", "8.8.8.8:53"}
	for _, target := range targets {
		d := net.Dialer{
			Timeout: timeout,
			Control: func(network, address string, c syscall.RawConn) error {
				var operr error
				fn := func(fd uintptr) {
					if devName != "" {
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
		start := time.Now()
		conn, err := d.DialContext(ctx, "tcp", target)
		if err == nil {
			_ = conn.Close()
			return true, max(1, int(time.Since(start).Milliseconds())), nil
		}
		if ctx.Err() != nil {
			return false, 0, ctx.Err()
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			unknown = true
		}
	}
	if unknown {
		return false, 0, context.DeadlineExceeded
	}
	return false, 0, nil
}
