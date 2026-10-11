package sstp

import (
	"bytes"
	"encoding/binary"
	"net"
	"testing"
)

func TestPacketHeaderAndChecksum(t *testing.T) {
	// 1. Test Control Packet Building
	attrs := []Attribute{
		{ID: AttrEncapsulatedProtocolID, Data: []byte{0x00, 0x01}},
	}
	ctrl := BuildControlPacket(MsgCallConnectRequest, attrs)
	if len(ctrl) != 14 {
		t.Fatalf("expected ctrl len 14, got %d", len(ctrl))
	}
	if ctrl[0] != SSTPVersion || ctrl[1] != SSTPFlagControl {
		t.Fatalf("expected sstp control header, got %x %x", ctrl[0], ctrl[1])
	}
	msgType := binary.BigEndian.Uint16(ctrl[4:6])
	if msgType != MsgCallConnectRequest {
		t.Fatalf("expected MsgCallConnectRequest, got %d", msgType)
	}

	// 2. Test Checksum Calculation
	data := []byte{0x45, 0x00, 0x00, 0x3c, 0x1c, 0x46, 0x40, 0x00, 0x40, 0x06, 0x00, 0x00, 0xac, 0x10, 0x0a, 0x63, 0xac, 0x10, 0x0a, 0x0c}
	chk := InternetChecksum(data, 0, len(data))
	if chk == 0 {
		t.Fatalf("checksum should not be zero")
	}

	// Odd length checksum
	oddData := []byte{0x01, 0x02, 0x03}
	oddChk := InternetChecksum(oddData, 0, len(oddData))
	if oddChk == 0 {
		t.Fatalf("odd checksum should not be zero")
	}
}

func TestIPv4TCPPacketRoundTrip(t *testing.T) {
	srcIP := net.ParseIP("10.211.1.2")
	dstIP := net.ParseIP("1.1.1.1")
	payload := []byte("GET / HTTP/1.1\r\nHost: 1.1.1.1\r\n\r\n")

	rawPacket := BuildIPv4TCPPacket(
		srcIP, dstIP,
		45000, 80,
		1001, 2002,
		TCPFlagACK|TCPFlagPSH,
		payload,
	)
	if rawPacket == nil {
		t.Fatalf("failed to build IPv4 TCP packet")
	}

	// SSTP Data Header (6 bytes) + PPP Header (2 bytes) = 8 bytes prefix
	if len(rawPacket) < 8+40+len(payload) {
		t.Fatalf("packet too short: %d", len(rawPacket))
	}
	if rawPacket[0] != SSTPVersion || rawPacket[1] != SSTPFlagData {
		t.Fatalf("expected sstp data flags")
	}
	if rawPacket[4] != 0xff || rawPacket[5] != 0x03 {
		t.Fatalf("expected HDLC framing")
	}

	pppProto := binary.BigEndian.Uint16(rawPacket[6:8])
	if pppProto != ProtoIPv4 {
		t.Fatalf("expected ProtoIPv4, got 0x%04x", pppProto)
	}

	ipBytes := rawPacket[8:]
	parsed, err := ParseIncomingIPv4TCP(ipBytes)
	if err != nil {
		t.Fatalf("failed to parse IPv4 TCP packet: %v", err)
	}

	if !parsed.SrcIP.Equal(srcIP) {
		t.Fatalf("expected src ip %s, got %s", srcIP, parsed.SrcIP)
	}
	if !parsed.DstIP.Equal(dstIP) {
		t.Fatalf("expected dst ip %s, got %s", dstIP, parsed.DstIP)
	}
	if parsed.SrcPort != 45000 || parsed.DstPort != 80 {
		t.Fatalf("expected ports 45000->80, got %d->%d", parsed.SrcPort, parsed.DstPort)
	}
	if parsed.Seq != 1001 || parsed.Ack != 2002 {
		t.Fatalf("expected seq/ack 1001/2002, got %d/%d", parsed.Seq, parsed.Ack)
	}
	if parsed.Flags != (TCPFlagACK | TCPFlagPSH) {
		t.Fatalf("expected flags 0x18, got 0x%02x", parsed.Flags)
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Fatalf("payload mismatch: %s != %s", parsed.Payload, payload)
	}
}

func TestPPPLCPAndPAPPackets(t *testing.T) {
	// LCP Config-Request with MRU 1500
	lcp := BuildLCPPacket(PPPCodeConfigureRequest, 1, []byte{LCPOptMRU, 4, 0x05, 0xdc})
	if len(lcp) != 10 {
		t.Fatalf("expected LCP length 10, got %d", len(lcp))
	}
	if binary.BigEndian.Uint16(lcp[0:2]) != ProtoLCP {
		t.Fatalf("expected ProtoLCP")
	}

	// PAP Auth-Request
	pap := BuildPAPAuthRequest(2, "vpn", "vpn")
	if binary.BigEndian.Uint16(pap[0:2]) != ProtoPAP {
		t.Fatalf("expected ProtoPAP")
	}
	if pap[2] != PAPCodeAuthRequest || pap[3] != 2 {
		t.Fatalf("unexpected PAP code/id")
	}
}

func TestIPv4UDPPacketRoundTrip(t *testing.T) {
	srcIP := net.ParseIP("10.211.1.2")
	dstIP := net.ParseIP("8.8.8.8")
	payload := []byte{0x12, 0x34, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00} // DNS query stub

	rawPacket := BuildIPv4UDPPacket(
		srcIP, dstIP,
		53123, 53,
		payload,
	)
	if rawPacket == nil {
		t.Fatalf("failed to build IPv4 UDP packet")
	}

	// SSTP Data Header (6 bytes) + PPP Header (2 bytes) = 8 bytes prefix
	if len(rawPacket) < 8+28+len(payload) {
		t.Fatalf("packet too short: %d", len(rawPacket))
	}

	pppProto := binary.BigEndian.Uint16(rawPacket[6:8])
	if pppProto != ProtoIPv4 {
		t.Fatalf("expected ProtoIPv4, got 0x%04x", pppProto)
	}

	ipBytes := rawPacket[8:]
	parsed, err := ParseIncomingIPv4UDP(ipBytes)
	if err != nil {
		t.Fatalf("failed to parse IPv4 UDP packet: %v", err)
	}

	if !parsed.SrcIP.Equal(srcIP) {
		t.Fatalf("expected src ip %s, got %s", srcIP, parsed.SrcIP)
	}
	if !parsed.DstIP.Equal(dstIP) {
		t.Fatalf("expected dst ip %s, got %s", dstIP, parsed.DstIP)
	}
	if parsed.SrcPort != 53123 || parsed.DstPort != 53 {
		t.Fatalf("expected ports 53123->53, got %d->%d", parsed.SrcPort, parsed.DstPort)
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Fatalf("payload mismatch")
	}
}
