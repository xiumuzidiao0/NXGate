package proxy

import (
	"encoding/binary"
	"fmt"
	"net"
)

// SOCKS5 UDP packet format:
// +----+------+------+----------+----------+----------+
// |RSV | FRAG | ATYP | DST.ADDR | DST.PORT |   DATA   |
// +----+------+------+----------+----------+----------+
// | 2  |  1   |  1   | Variable |    2     | Variable |
// +----+------+------+----------+----------+----------+

const (
	socks5UDPReserved = 0x0000
	socks5FragNo      = 0x00 // We don't support fragmentation
)

// parseSocks5UDPRequest extracts target address and payload from a SOCKS5 UDP packet
func parseSocks5UDPRequest(data []byte) (addr string, payload []byte, err error) {
	if len(data) < 10 {
		return "", nil, fmt.Errorf("UDP packet too short: %d bytes", len(data))
	}

	// RSV (2 bytes) - must be 0x0000
	if data[0] != 0x00 || data[1] != 0x00 {
		return "", nil, fmt.Errorf("invalid RSV field: %02x%02x", data[0], data[1])
	}

	// FRAG (1 byte) - we only support 0x00 (no fragmentation)
	if data[2] != socks5FragNo {
		return "", nil, fmt.Errorf("fragmentation not supported: FRAG=%02x", data[2])
	}

	// ATYP (1 byte)
	atyp := data[3]
	pos := 4

	var host string
	var port uint16

	switch atyp {
	case atypIPv4: // IPv4
		if len(data) < pos+6 {
			return "", nil, fmt.Errorf("incomplete IPv4 address")
		}
		host = net.IP(data[pos : pos+4]).String()
		pos += 4

	case atypDomain: // Domain name
		if len(data) < pos+1 {
			return "", nil, fmt.Errorf("incomplete domain length")
		}
		domainLen := int(data[pos])
		pos++
		if len(data) < pos+domainLen+2 {
			return "", nil, fmt.Errorf("incomplete domain name")
		}
		host = string(data[pos : pos+domainLen])
		pos += domainLen

	case atypIPv6: // IPv6
		if len(data) < pos+18 {
			return "", nil, fmt.Errorf("incomplete IPv6 address")
		}
		host = net.IP(data[pos : pos+16]).String()
		pos += 16

	default:
		return "", nil, fmt.Errorf("unsupported address type: 0x%02x", atyp)
	}

	// Port (2 bytes, big-endian)
	port = binary.BigEndian.Uint16(data[pos : pos+2])
	pos += 2

	addr = fmt.Sprintf("%s:%d", host, port)
	payload = data[pos:]
	return addr, payload, nil
}

// buildSocks5UDPResponse wraps payload into a SOCKS5 UDP response packet
func buildSocks5UDPResponse(fromAddr *net.UDPAddr, payload []byte) ([]byte, error) {
	ip4 := fromAddr.IP.To4()
	ip6 := fromAddr.IP.To16()

	var buf []byte

	// RSV + FRAG
	buf = append(buf, 0x00, 0x00, socks5FragNo)

	if ip4 != nil {
		// ATYP = IPv4
		buf = append(buf, atypIPv4)
		buf = append(buf, ip4...)
	} else if ip6 != nil {
		// ATYP = IPv6
		buf = append(buf, atypIPv6)
		buf = append(buf, ip6...)
	} else {
		return nil, fmt.Errorf("invalid source address: %v", fromAddr)
	}

	// Port (big-endian)
	portBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(portBytes, uint16(fromAddr.Port))
	buf = append(buf, portBytes...)

	// Payload
	buf = append(buf, payload...)

	return buf, nil
}
