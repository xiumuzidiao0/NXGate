package sstp

import (
	"bytes"
	"net"
	"testing"
	"time"
)

type mockRawSender struct {
	sentPackets [][]byte
}

func TestUDPConnReadWrite(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("8.8.8.8")
	tun := &Tunnel{
		assignedIP: localIP,
		udpConns:   make(map[uint16]*UDPConn),
	}

	uConn := newUDPConn(tun, localIP, 54321)
	tun.udpConns[54321] = uConn

	// 1. Simulate incoming UDP packet
	payload := []byte("dns response payload")
	incoming := &ParsedUDPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 53,
		DstPort: 54321,
		Payload: payload,
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		uConn.handleIncoming(incoming)
	}()

	buf := make([]byte, 1024)
	n, addr, err := uConn.ReadFrom(buf)
	if err != nil {
		t.Fatalf("ReadFrom failed: %v", err)
	}
	if n != len(payload) || !bytes.Equal(buf[:n], payload) {
		t.Fatalf("unexpected data: %s", buf[:n])
	}
	udpAddr, ok := addr.(*net.UDPAddr)
	if !ok || !udpAddr.IP.Equal(remoteIP) || udpAddr.Port != 53 {
		t.Fatalf("unexpected remote addr: %v", addr)
	}

	// 2. Test LocalAddr
	lAddr := uConn.LocalAddr().(*net.UDPAddr)
	if !lAddr.IP.Equal(localIP) || lAddr.Port != 54321 {
		t.Fatalf("unexpected local addr: %v", lAddr)
	}

	// 3. Test Deadline Timeout
	err = uConn.SetReadDeadline(time.Now().Add(20 * time.Millisecond))
	if err != nil {
		t.Fatalf("SetReadDeadline failed: %v", err)
	}
	_, _, err = uConn.ReadFrom(buf)
	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}

	// 4. Test Close
	_ = uConn.Close()
	_, _, err = uConn.ReadFrom(buf)
	if err == nil {
		t.Fatalf("expected error on closed connection, got nil")
	}
}
