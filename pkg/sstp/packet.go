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

func tcpChecksum(srcIP, dstIP []byte, tcpSeg []byte) uint16 {
	var sum uint32
	sum += uint32(binary.BigEndian.Uint16(srcIP[0:2]))
	sum += uint32(binary.BigEndian.Uint16(srcIP[2:4]))
	sum += uint32(binary.BigEndian.Uint16(dstIP[0:2]))
	sum += uint32(binary.BigEndian.Uint16(dstIP[2:4]))
	sum += uint32(6) // TCP
	sum += uint32(len(tcpSeg))

	end := len(tcpSeg)
	for i := 0; i < end-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(tcpSeg[i : i+2]))
	}
	if end%2 == 1 {
		sum += uint32(tcpSeg[end-1]) << 8
	}

	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}

func udpChecksum(srcIP, dstIP []byte, udpSeg []byte) uint16 {
	var sum uint32
	sum += uint32(binary.BigEndian.Uint16(srcIP[0:2]))
	sum += uint32(binary.BigEndian.Uint16(srcIP[2:4]))
	sum += uint32(binary.BigEndian.Uint16(dstIP[0:2]))
	sum += uint32(binary.BigEndian.Uint16(dstIP[2:4]))
	sum += uint32(17) // UDP
	sum += uint32(len(udpSeg))

	end := len(udpSeg)
	for i := 0; i < end-1; i += 2 {
		sum += uint32(binary.BigEndian.Uint16(udpSeg[i : i+2]))
	}
	if end%2 == 1 {
		sum += uint32(udpSeg[end-1]) << 8
	}

	for sum > 0xffff {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	chk := ^uint16(sum)
	if chk == 0 {
		return 0xffff
	}
	return chk
}

func BuildIPv4TCPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, seq, ack uint32, flags byte, payload []byte) []byte {
	sIPv4 := srcIP.To4()
	dIPv4 := dstIP.To4()
	if sIPv4 == nil || dIPv4 == nil {
		return nil
	}

	totalLen := 48 + len(payload)
	pkt := make([]byte, totalLen)

	// 1. SSTP Data Header (6 bytes)
	pkt[0] = SSTPVersion
	pkt[1] = SSTPFlagData
	binary.BigEndian.PutUint16(pkt[2:4], uint16(totalLen)|0x8000)
	pkt[4] = 0xff
	pkt[5] = 0x03

	// 2. PPP Protocol (2 bytes): 0x0021 IPv4
	binary.BigEndian.PutUint16(pkt[6:8], ProtoIPv4)

	// 3. IPv4 Header (20 bytes: pkt[8:28])
	ipLen := uint16(40 + len(payload))
	pkt[8] = 0x45 // Version 4, IHL 5
	pkt[9] = 0x00
	binary.BigEndian.PutUint16(pkt[10:12], ipLen)
	binary.BigEndian.PutUint16(pkt[12:14], uint16(rand.Intn(60000)+1000))
	binary.BigEndian.PutUint16(pkt[14:16], 0x4000) // DF flag
	pkt[16] = 64                                  // TTL
	pkt[17] = 6                                   // Protocol TCP
	copy(pkt[20:24], sIPv4)
	copy(pkt[24:28], dIPv4)
	ipChk := InternetChecksum(pkt[8:28], 0, 20)
	binary.BigEndian.PutUint16(pkt[18:20], ipChk)

	// 4. TCP Header (20 bytes: pkt[28:48]) + Payload (pkt[48:])
	binary.BigEndian.PutUint16(pkt[28:30], srcPort)
	binary.BigEndian.PutUint16(pkt[30:32], dstPort)
	binary.BigEndian.PutUint32(pkt[32:36], seq)
	binary.BigEndian.PutUint32(pkt[36:40], ack)
	pkt[40] = 0x50 // Data offset 5 (20 bytes)
	pkt[41] = flags
	binary.BigEndian.PutUint16(pkt[42:44], 65535) // Window size
	// Checksum placeholder at pkt[44:46] is 0

	if len(payload) > 0 {
		copy(pkt[48:], payload)
	}

	tcpChk := tcpChecksum(sIPv4, dIPv4, pkt[28:])
	binary.BigEndian.PutUint16(pkt[44:46], tcpChk)

	return pkt
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

type ParsedUDPPacket struct {
	SrcIP   net.IP
	DstIP   net.IP
	SrcPort uint16
	DstPort uint16
	Payload []byte
}

func BuildIPv4UDPPacket(srcIP, dstIP net.IP, srcPort, dstPort uint16, payload []byte) []byte {
	sIPv4 := srcIP.To4()
	dIPv4 := dstIP.To4()
	if sIPv4 == nil || dIPv4 == nil {
		return nil
	}

	totalLen := 36 + len(payload)
	pkt := make([]byte, totalLen)

	// 1. SSTP Data Header (6 bytes)
	pkt[0] = SSTPVersion
	pkt[1] = SSTPFlagData
	binary.BigEndian.PutUint16(pkt[2:4], uint16(totalLen)|0x8000)
	pkt[4] = 0xff
	pkt[5] = 0x03

	// 2. PPP Protocol (2 bytes): 0x0021 IPv4
	binary.BigEndian.PutUint16(pkt[6:8], ProtoIPv4)

	// 3. IPv4 Header (20 bytes: pkt[8:28])
	ipLen := uint16(28 + len(payload))
	pkt[8] = 0x45 // Version 4, IHL 5
	pkt[9] = 0x00
	binary.BigEndian.PutUint16(pkt[10:12], ipLen)
	binary.BigEndian.PutUint16(pkt[12:14], uint16(rand.Intn(60000)+1000))
	binary.BigEndian.PutUint16(pkt[14:16], 0x4000) // DF flag
	pkt[16] = 64                                  // TTL
	pkt[17] = 17                                  // Protocol UDP
	copy(pkt[20:24], sIPv4)
	copy(pkt[24:28], dIPv4)
	ipChk := InternetChecksum(pkt[8:28], 0, 20)
	binary.BigEndian.PutUint16(pkt[18:20], ipChk)

	// 4. UDP Header (8 bytes: pkt[28:36]) + Payload (pkt[36:])
	udpLen := uint16(8 + len(payload))
	binary.BigEndian.PutUint16(pkt[28:30], srcPort)
	binary.BigEndian.PutUint16(pkt[30:32], dstPort)
	binary.BigEndian.PutUint16(pkt[32:34], udpLen)
	// Checksum placeholder at pkt[34:36] is 0

	if len(payload) > 0 {
		copy(pkt[36:], payload)
	}

	udpChk := udpChecksum(sIPv4, dIPv4, pkt[28:])
	binary.BigEndian.PutUint16(pkt[34:36], udpChk)

	return pkt
}

func ParseIncomingIPv4UDP(ipPacket []byte) (*ParsedUDPPacket, error) {
	if len(ipPacket) < 28 { // 20 IP + 8 UDP
		return nil, errors.New("packet too short")
	}
	if ipPacket[0]>>4 != 4 {
		return nil, errors.New("not IPv4")
	}
	ipHdrLen := int(ipPacket[0]&0x0f) * 4
	if len(ipPacket) < ipHdrLen+8 {
		return nil, errors.New("truncated IPv4 header")
	}
	if ipPacket[9] != 17 { // UDP
		return nil, errors.New("not UDP protocol")
	}

	srcIP := net.IPv4(ipPacket[12], ipPacket[13], ipPacket[14], ipPacket[15])
	dstIP := net.IPv4(ipPacket[16], ipPacket[17], ipPacket[18], ipPacket[19])

	udpHdr := ipPacket[ipHdrLen:]
	srcPort := binary.BigEndian.Uint16(udpHdr[0:2])
	dstPort := binary.BigEndian.Uint16(udpHdr[2:4])
	udpLen := int(binary.BigEndian.Uint16(udpHdr[4:6]))
	if udpLen < 8 || len(udpHdr) < udpLen {
		return nil, errors.New("invalid or truncated UDP length")
	}
	payload := udpHdr[8:udpLen]

	return &ParsedUDPPacket{
		SrcIP:   srcIP,
		DstIP:   dstIP,
		SrcPort: srcPort,
		DstPort: dstPort,
		Payload: payload,
	}, nil
}
