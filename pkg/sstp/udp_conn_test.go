package sstp

import (
	"bytes"
	"net"
	"testing"
	"time"
)

func TestUDPConnReadWrite(t *testing.T) {
	bridge := newSSTPPeerBridge(t, sstpBridgeOpts{})
	defer bridge.Close()

	peerIP := net.ParseIP("8.8.8.8")
	srvUDP, err := bridge.listenPeerUDP(peerIP, 53)
	if err != nil {
		t.Fatalf("listenPeerUDP failed: %v", err)
	}
	defer srvUDP.Close()

	expected := []byte("dns response payload")
	go func() {
		buf := make([]byte, 512)
		n, from, rerr := srvUDP.ReadFrom(buf)
		if rerr != nil {
			return
		}
		if bytes.Equal(buf[:n], []byte("ping-query")) {
			_, _ = srvUDP.WriteTo(expected, from)
		}
	}()

	uConn, err := bridge.tun.OpenPacketConn()
	if err != nil {
		t.Fatalf("OpenPacketConn failed: %v", err)
	}

	// 1. Verify LocalAddr uses tunnel's assigned IPv4 and ephemeral port
	lAddr := uConn.LocalAddr().(*net.UDPAddr)
	if !lAddr.IP.Equal(bridge.tun.assignedIP) || lAddr.Port == 0 {
		t.Fatalf("unexpected local addr: %v", lAddr)
	}

	// 2. Round-trip UDP packet over SSTP + gVisor netstack
	if _, err := uConn.WriteTo([]byte("ping-query"), &net.UDPAddr{IP: peerIP, Port: 53}); err != nil {
		t.Fatalf("WriteTo failed: %v", err)
	}

	buf := make([]byte, 1024)
	_ = uConn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, addr, err := uConn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if n != len(expected) || !bytes.Equal(buf[:n], expected) {
		t.Fatalf("unexpected data: %q", buf[:n])
	}
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok || !udpAddr.IP.Equal(peerIP) || udpAddr.Port != 53 {
		t.Fatalf("unexpected remote addr: %v", addr)
	}

	// 3. Test Deadline Timeout
	if err := uConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond)); err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}
	if _, _, err := uConn.ReadFrom(buf); err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	// 4. Test Close
	_ = uConn.Close()
	if _, _, err := uConn.ReadFrom(buf); err == nil {
		t.Fatalf("expected error on closed connection, got nil")
	}
}
