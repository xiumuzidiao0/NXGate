package server

import (
	"net"
	"testing"
)

func TestWebListenAddress(t *testing.T) {
	for _, tc := range []struct{ host, wantHost, wantAddr string }{
		{"", "0.0.0.0", "0.0.0.0:8787"},
		{"0.0.0.0", "0.0.0.0", "0.0.0.0:8787"},
		{"127.0.0.1", "127.0.0.1", "127.0.0.1:8787"},
		{"[::1]", "::1", "[::1]:8787"},
	} {
		host, addr := webListenAddress(tc.host, 8787)
		if host != tc.wantHost || addr != tc.wantAddr {
			t.Errorf("%q: got host=%q addr=%q", tc.host, host, addr)
		}
	}
}

// Bind only an ephemeral test socket: no application startup, service, route or firewall changes.
func TestWebSocketRespectsDefaultAndExplicitLoopback(t *testing.T) {
	for _, configured := range []string{"", "127.0.0.1"} {
		_, addr := webListenAddress(configured, 0)
		ln, err := net.Listen("tcp4", addr)
		if err != nil {
			t.Fatal(err)
		}
		bound := ln.Addr().(*net.TCPAddr)
		if configured == "" && !bound.IP.IsUnspecified() {
			t.Errorf("default unexpectedly bound to %s", bound.IP)
		}
		if configured == "127.0.0.1" && !bound.IP.IsLoopback() {
			t.Errorf("explicit loopback changed to %s", bound.IP)
		}
		ln.Close()
	}
}
