package sstp

import (
	"encoding/binary"
	"errors"
	"math/rand"
	"net"
)

type Attribute struct {
	ID   byte
	Data []byte
}

func BuildControlPacket(msgType uint16, attrs []Attribute) []byte {
	attrLen := 0
	for _, a := range attrs {
		attrLen += 4 + len(a.Data)
	}
	pktLen := 8 + attrLen
	pkt := make([]byte, pktLen)
	pkt[0] = SSTPVersion
	pkt[1] = SSTPFlagControl
	binary.BigEndian.PutUint16(pkt[2:4], uint16(pktLen)|0x8000)
	binary.BigEndian.PutUint16(pkt[4:6], msgType)
	binary.BigEndian.PutUint16(pkt[6:8], uint16(len(attrs)))

	offset := 8
	for _, a := range attrs {
		pkt[offset] = 0 // reserved
		pkt[offset+1] = a.ID
		binary.BigEndian.PutUint16(pkt[offset+2:offset+4], uint16(4+len(a.Data)))
		copy(pkt[offset+4:], a.Data)
		offset += 4 + len(a.Data)
	}
	return pkt
}

func BuildDataPacket(pppFrame []byte) []byte {
	pktLen := 6 + len(pppFrame)
	pkt := make([]byte, pktLen)
	pkt[0] = SSTPVersion
	pkt[1] = SSTPFlagData
	binary.BigEndian.PutUint16(pkt[2:4], uint16(pktLen)|0x8000)
	pkt[4] = 0xff
	pkt[5] = 0x03
	copy(pkt[6:], pppFrame)
	return pkt
}

func BuildLCPPacket(code, ident byte, optionsData []byte) []byte {
	length := 4 + len(optionsData)
	buf := make([]byte, 2+length)
	binary.BigEndian.PutUint16(buf[0:2], ProtoLCP)
	buf[2] = code
	buf[3] = ident
	binary.BigEndian.PutUint16(buf[4:6], uint16(length))
	copy(buf[6:], optionsData)
	return buf
}

func BuildPAPAuthRequest(ident byte, user, pass string) []byte {
	uBytes := []byte(user)
	pBytes := []byte(pass)
	dataLen := 1 + len(uBytes) + 1 + len(pBytes)
	totLen := 4 + dataLen

	buf := make([]byte, 2+totLen)
	binary.BigEndian.PutUint16(buf[0:2], ProtoPAP)
	buf[2] = PAPCodeAuthRequest
	buf[3] = ident
	binary.BigEndian.PutUint16(buf[4:6], uint16(totLen))
	buf[6] = byte(len(uBytes))
	copy(buf[7:], uBytes)
	buf[7+len(uBytes)] = byte(len(pBytes))
	copy(buf[8+len(uBytes):], pBytes)
	return buf
}

func BuildIPCPPacket(code, ident byte, optionsData []byte) []byte {
	length := 4 + len(optionsData)
	buf := make([]byte, 2+length)
	binary.BigEndian.PutUint16(buf[0:2], ProtoIPCP)
	buf[2] = code
	buf[3] = ident
	binary.BigEndian.PutUint16(buf[4:6], uint16(length))
	copy(buf[6:], optionsData)
	return buf
}

func InternetChecksum(data []byte, offset, length int) uint16 {
	var sum uint32
	end := offset + length
	for i := offset; i < end-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(data[i : i+2]))
	}
	if length%2 == 1 {
		sum += uint32(data[end-1]) << 8
	}
	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

func BuildIPv4TCPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, seq, ack uint32, flags byte, payload []byte) []byte {
	sIPv4 := srcIP.To4()
	dIPv4 := dstIP.To4()
	if sIPv4 == nil || dIPv4 == nil {
		return nil
	}

	tcpLen := 20 + len(payload)
	ipLen := 20 + tcpLen

	// IP Header (20 bytes)
	ipHdr := make([]byte, 20)
	ipHdr[0] = 0x45 // Version 4, IHL 5
	ipHdr[1] = 0x00 // DSCP / ECN
	binary.BigEndian.PutUint16(ipHdr[2:4], uint16(ipLen))
	binary.BigEndian.PutUint16(ipHdr[4:6], uint16(rand.Intn(60000)+1000))
	binary.BigEndian.PutUint16(ipHdr[6:8], 0x4000) // DF flag
	ipHdr[8] = 64                                  // TTL
	ipHdr[9] = 6                                   // Protocol TCP
	copy(ipHdr[12:16], sIPv4)
	copy(ipHdr[16:20], dIPv4)
	ipChk := InternetChecksum(ipHdr, 0, 20)
	binary.BigEndian.PutUint16(ipHdr[10:12], ipChk)

	// TCP Pseudo Header + TCP Segment for Checksum
	pseudo := make([]byte, 12+tcpLen)
	copy(pseudo[0:4], sIPv4)
	copy(pseudo[4:8], dIPv4)
	pseudo[9] = 6 // TCP
	binary.BigEndian.PutUint16(pseudo[10:12], uint16(tcpLen))

	// TCP Header
	tcpSeg := pseudo[12:]
	binary.BigEndian.PutUint16(tcpSeg[0:2], srcPort)
	binary.BigEndian.PutUint16(tcpSeg[2:4], dstPort)
	binary.BigEndian.PutUint32(tcpSeg[4:8], seq)
	binary.BigEndian.PutUint32(tcpSeg[8:12], ack)
	tcpSeg[12] = 0x50 // Data offset 5 (20 bytes), reserved 0
	tcpSeg[13] = flags
	binary.BigEndian.PutUint16(tcpSeg[14:16], 65535) // Window size
	if len(payload) > 0 {
		copy(tcpSeg[20:], payload)
	}

	tcpChk := InternetChecksum(pseudo, 0, len(pseudo))
	binary.BigEndian.PutUint16(tcpSeg[16:18], tcpChk)

	// Combine into full IPv4 packet
	fullIP := make([]byte, ipLen)
	copy(fullIP[:20], ipHdr)
	copy(fullIP[20:], tcpSeg)

	// PPP framing (protocol 0x0021 = IPv4)
	pppFrame := make([]byte, 2+ipLen)
	binary.BigEndian.PutUint16(pppFrame[0:2], ProtoIPv4)
	copy(pppFrame[2:], fullIP)

	return BuildDataPacket(pppFrame)
}

type ParsedTCPPacket struct {
	SrcIP    net.IP
	DstIP    net.IP
	SrcPort  uint16
	DstPort  uint16
	Seq      uint32
	Ack      uint32
	Flags    byte
	Payload  []byte
}

func ParseIncomingIPv4TCP(ipPacket []byte) (*ParsedTCPPacket, error) {
	if len(ipPacket) < 40 {
		return nil, errors.New("packet too short")
	}
	if ipPacket[0]>>4 != 4 {
		return nil, errors.New("not IPv4")
	}
	ipHdrLen := int(ipPacket[0]&0x0f) * 4
	if len(ipPacket) < ipHdrLen+20 {
		return nil, errors.New("truncated IPv4 header")
	}
	if ipPacket[9] != 6 { // TCP
		return nil, errors.New("not TCP protocol")
	}

	srcIP := net.IPv4(ipPacket[12], ipPacket[13], ipPacket[14], ipPacket[15])
	dstIP := net.IPv4(ipPacket[16], ipPacket[17], ipPacket[18], ipPacket[19])

	tcpHdr := ipPacket[ipHdrLen:]
	srcPort := binary.BigEndian.Uint16(tcpHdr[0:2])
	dstPort := binary.BigEndian.Uint16(tcpHdr[2:4])
	seq := binary.BigEndian.Uint32(tcpHdr[4:8])
	ack := binary.BigEndian.Uint32(tcpHdr[8:12])
	dataOffset := int((tcpHdr[12]>>4)&0x0f) * 4
	if len(tcpHdr) < dataOffset {
		return nil, errors.New("truncated TCP header")
	}
	flags := tcpHdr[13]
	payload := tcpHdr[dataOffset:]

	return &ParsedTCPPacket{
		SrcIP:   srcIP,
		DstIP:   dstIP,
		SrcPort: srcPort,
		DstPort: dstPort,
		Seq:     seq,
		Ack:     ack,
		Flags:   flags,
		Payload: payload,
	}, nil
}
