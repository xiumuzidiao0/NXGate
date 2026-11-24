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

func TestTCPConnWriteDeadline(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45000, 80, 1000, 2000)
	if err := conn.SetWriteDeadline(time.Now().Add(-time.Second)); err != nil {
		t.Fatalf("SetWriteDeadline failed: %v", err)
	}
	if _, err := conn.Write([]byte("hello")); err != ErrTimeout {
		t.Fatalf("expected ErrTimeout on expired write deadline, got %v", err)
	}
}

func TestTCPConnBoundedBufferAndWindowBackpressure(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45001, 80, 1000, 2000)

	conn.mu.Lock()
	if conn.outOfOrder != nil {
		conn.mu.Unlock()
		t.Fatalf("expected lazy nil outOfOrder map on new TCPConn")
	}
	conn.mu.Unlock()

	// Fill receive buffer up to maxTCPReadBufCap.
	seq := uint32(2000)
	chunk := bytes.Repeat([]byte("A"), DefaultTCPMSS)
	remaining := maxTCPReadBufCap
	for remaining > 0 {
		sz := min(len(chunk), remaining)
		conn.handleIncoming(&ParsedTCPPacket{
			SrcIP:   remoteIP,
			DstIP:   localIP,
			SrcPort: 80,
			DstPort: 45001,
			Seq:     seq,
			Ack:     1000,
			Flags:   TCPFlagACK,
			Payload: chunk[:sz],
		})
		seq += uint32(sz)
		remaining -= sz
	}

	conn.mu.Lock()
	bufLen := conn.readBuf.Len()
	wnd := conn.advertisedWindowLocked()
	zeroSent := conn.zeroWindowSent
	conn.mu.Unlock()

	if bufLen != maxTCPReadBufCap || wnd != 0 || !zeroSent {
		t.Fatalf("expected full bounded buffer: len=%d wnd=%d zeroSent=%v", bufLen, wnd, zeroSent)
	}

	// Extra bytes when buffer is full must be dropped without advancing ACK.
	overflow := []byte("OVERFLOW-100")
	conn.handleIncoming(&ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45001,
		Seq:     seq,
		Ack:     1000,
		Flags:   TCPFlagACK,
		Payload: overflow,
	})
	if conn.ack != seq {
		t.Fatalf("expected ACK %d unchanged on full buffer, got %d", seq, conn.ack)
	}

	// Drain 2*DefaultTCPMSS via Read; zero-window state must reopen.
	drain := make([]byte, 2*DefaultTCPMSS)
	if _, err := io.ReadFull(conn, drain); err != nil {
		t.Fatalf("drain read failed: %v", err)
	}

	conn.mu.Lock()
	reopened := !conn.zeroWindowSent && conn.advertisedWindowLocked() >= 2*DefaultTCPMSS
	conn.mu.Unlock()
	if !reopened {
		t.Fatalf("expected send window to reopen after draining >= 2*MSS")
	}

	// Retransmitted segment now fits cleanly.
	conn.handleIncoming(&ParsedTCPPacket{
		SrcIP:   remoteIP,
		DstIP:   localIP,
		SrcPort: 80,
		DstPort: 45001,
		Seq:     seq,
		Ack:     1000,
		Flags:   TCPFlagACK,
		Payload: overflow,
	})
	if conn.ack != seq+uint32(len(overflow)) {
		t.Fatalf("expected ACK %d after reopen, got %d", seq+uint32(len(overflow)), conn.ack)
	}

	_ = conn.Close()
	conn.mu.Lock()
	released := conn.readBuf.buf == nil && conn.readBuf.Len() == 0 && conn.outOfOrder == nil
	conn.mu.Unlock()
	if !released {
		t.Fatalf("expected Close to release ring buffer and outOfOrder map")
	}
}

func TestTCPConnLazyOutOfOrderAccounting(t *testing.T) {
	localIP := net.ParseIP("10.211.1.2")
	remoteIP := net.ParseIP("1.1.1.1")
	tun := &Tunnel{
		assignedIP: localIP,
		sessions:   make(map[uint32]*TCPConn),
	}

	conn := newTCPConn(tun, localIP, remoteIP, 45002, 80, 1000, 2000)

	// In-order write must keep outOfOrder nil.
	conn.handleIncoming(&ParsedTCPPacket{
		SrcIP: remoteIP, DstIP: localIP, SrcPort: 80, DstPort: 45002,
		Seq: 2000, Ack: 1000, Flags: TCPFlagACK, Payload: []byte("AB"),
	})
	conn.mu.Lock()
	if conn.outOfOrder != nil {
		conn.mu.Unlock()
		t.Fatalf("expected nil outOfOrder after in-order segment")
	}
	conn.mu.Unlock()

	// Out-of-order segment initializes outOfOrder and tracks bytes; duplicate is ignored.
	ooo := &ParsedTCPPacket{
		SrcIP: remoteIP, DstIP: localIP, SrcPort: 80, DstPort: 45002,
		Seq: 2004, Ack: 1000, Flags: TCPFlagACK, Payload: []byte("EFG"),
	}
	conn.handleIncoming(ooo)
	conn.handleIncoming(ooo)
	conn.mu.Lock()
	if conn.outOfOrder == nil || conn.outOfOrderBytes != 3 {
		conn.mu.Unlock()
		t.Fatalf("expected outOfOrderBytes=3, got %d (map=%v)", conn.outOfOrderBytes, conn.outOfOrder)
	}
	conn.mu.Unlock()

	// Fill gap and confirm outOfOrderBytes drops back to 0.
	conn.handleIncoming(&ParsedTCPPacket{
		SrcIP: remoteIP, DstIP: localIP, SrcPort: 80, DstPort: 45002,
		Seq: 2002, Ack: 1000, Flags: TCPFlagACK, Payload: []byte("CD"),
	})
	conn.mu.Lock()
	remOOO := conn.outOfOrderBytes
	conn.mu.Unlock()
	if remOOO != 0 {
		t.Fatalf("expected outOfOrderBytes=0 after hole fill, got %d", remOOO)
	}

	got := make([]byte, 7)
	if _, err := io.ReadFull(conn, got); err != nil || string(got) != "ABCDEFG" {
		t.Fatalf("expected ABCDEFG, got %q err=%v", string(got), err)
	}
}
