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

func TestTCPConnOutOfOrderReassembly(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45000, 80, 1000, 2000)

	// 1. Send segment 2 FIRST (Out of Order: Seq=2005)
	seg2 := &ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45000,
		Seq:     2005,
		Ack:     1000,
		Flags:   TCPFlagACK,
		Payload: []byte("WORLD"),
	}
	conn.handleIncoming(seg2)

	// Verify nothing is readable yet (hole at 2000..2005)
	conn.mu.Lock()
	bufferedLen := conn.readBuf.Len()
	conn.mu.Unlock()
	if bufferedLen != 0 {
		t.Fatalf("expected 0 readable bytes while waiting for segment 1, got %d", bufferedLen)
	}

	// 2. Send segment 1 (Fills the hole: Seq=2000)
	seg1 := &ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45000,
		Seq:     2000,
		Ack:     1000,
		Flags:   TCPFlagACK,
		Payload: []byte("HELLO"),
	}
	conn.handleIncoming(seg1)

	// 3. Read accumulated stream: should be "HELLOWORLD" in perfect order
	buf := make([]byte, 10)
	n, err := io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if n != 10 || string(buf) != "HELLOWORLD" {
		t.Fatalf("expected HELLOWORLD, got %s", string(buf[:n]))
	}

	// Verify ack advanced to 2010
	if conn.ack != 2010 {
		t.Fatalf("expected ack 2010, got %d", conn.ack)
	}
}

func TestTCPConnDuplicatePacketSuppression(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45000, 80, 1000, 2000)

	// 1. Send packet once
	pkt := &ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45000,
		Seq:     2000,
		Ack:     1000,
		Flags:   TCPFlagACK,
		Payload: []byte("DATA"),
	}
	conn.handleIncoming(pkt)

	buf := make([]byte, 4)
	n, err := io.ReadFull(conn, buf)
	if err != nil || n != 4 || string(buf) != "DATA" {
		t.Fatalf("first read failed")
	}

	// 2. Send identical packet again (retransmitted duplicate)
	conn.handleIncoming(pkt)

	conn.mu.Lock()
	remLen := conn.readBuf.Len()
	conn.mu.Unlock()
	if remLen != 0 {
		t.Fatalf("expected duplicate packet to be discarded, but buffer has %d bytes", remLen)
	}
}

func BenchmarkTCPConnThroughput(b *testing.B) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45000, 80, 1000, 2000)
	chunk := make([]byte, DefaultTCPMSS)
	readBuf := make([]byte, DefaultTCPMSS)

	b.SetBytes(int64(len(chunk)))
	b.ReportAllocs()
	b.ResetTimer()

	seq := uint32(2000)
	for i := 0; i < b.N; i++ {
		pkt := &ParsedTCPPacket{
			SrcIP:   remoteIP,
			DstIP:   localIP,
			SrcPort: 80,
			DstPort: 45000,
			Seq:     seq,
			Ack:     1000,
			Flags:   TCPFlagACK,
			Payload: chunk,
		}
		conn.handleIncoming(pkt)
		seq += uint32(len(chunk))
		_, _ = io.ReadFull(conn, readBuf)
	}
}
