package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"syscall"
	"time"
)

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
	if ctx.Err() != nil {
		return false, ctx.Err()
	}
	if devName == "" {
		return false, fmt.Errorf("no tunnel device")
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
		conn, err := d.DialContext(ctx, "tcp", target)
		if err == nil {
			_ = conn.Close()
			return true, nil
		}
		if ctx.Err() != nil {
			return false, ctx.Err()
		}
		var ne net.Error
		if errors.As(err, &ne) && ne.Timeout() {
			unknown = true
		}
	}
	if unknown {
		return false, context.DeadlineExceeded
	}
	return false, nil
}
