package sstp

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

type dummyTunnel struct {
	sentPackets [][]byte
}

func TestTCPConnReadWrite(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45000, 80, 1000, 2000)

	// Simulate incoming data packet
	payload := []byte("HTTP/1.1 200 OK\r\nContent-Length: 5\r\n\r\nhello")
	incoming := &ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45000,
		Seq:     2000,
		Ack:     1000,
		Flags:   TCPFlagACK | TCPFlagPSH,
		Payload: payload,
	}

	go func() {
		time.Sleep(10 * time.Millisecond)
		conn.handleIncoming(incoming)
	}()

	buf := make([]byte, len(payload))
	n, err := io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if n != len(payload) || !bytes.Equal(buf, payload) {
		t.Fatalf("unexpected read data: %s", buf[:n])
	}

	// Verify ack advanced
	if conn.ack != 2000+uint32(len(payload)) {
		t.Fatalf("expected ack %d, got %d", 2000+uint32(len(payload)), conn.ack)
	}

	// Test peer FIN
	finIncoming := &ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45000,
		Seq:     conn.ack,
		Ack:     1000,
		Flags:   TCPFlagACK | TCPFlagFIN,
	}
	conn.handleIncoming(finIncoming)

	// Next read should return EOF
	var dummy [10]byte
	_, err = conn.Read(dummy[:])
	if err != io.EOF {
		t.Fatalf("expected EOF after remote FIN, got: %v", err)
	}
}
