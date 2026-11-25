package sstp

import "encoding/binary"

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
