package sstp

import (
	"encoding/binary"
	"testing"
)

func TestSSTPControlAndDataPackets(t *testing.T) {
	// 1. Control Packet Building
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
	if got := binary.BigEndian.Uint16(ctrl[4:6]); got != MsgCallConnectRequest {
		t.Fatalf("expected MsgCallConnectRequest, got %d", got)
	}

	// 2. SSTP Data Packet wrapping PPP frame
	lcp := BuildLCPPacket(PPPCodeEchoRequest, 9, []byte{1, 2, 3, 4})
	pkt := BuildDataPacket(lcp)
	if pkt[0] != SSTPVersion || pkt[1] != SSTPFlagData {
		t.Fatalf("expected sstp data flags, got %x %x", pkt[0], pkt[1])
	}
	if pkt[4] != 0xff || pkt[5] != 0x03 {
		t.Fatalf("expected HDLC framing, got %x %x", pkt[4], pkt[5])
	}
	if got := int(binary.BigEndian.Uint16(pkt[2:4]) & 0x0fff); got != len(pkt) {
		t.Fatalf("expected length %d, got %d", len(pkt), got)
	}

	// 3. IPCP packet encoding
	ipcp := BuildIPCPPacket(PPPCodeConfigureRequest, 3, []byte{IPCPOptIPAddress, 6, 10, 0, 0, 1})
	if binary.BigEndian.Uint16(ipcp[0:2]) != ProtoIPCP || ipcp[2] != PPPCodeConfigureRequest || ipcp[3] != 3 {
		t.Fatalf("unexpected IPCP header: %x", ipcp)
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
