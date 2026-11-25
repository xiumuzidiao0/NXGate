package sstp

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strings"
	"time"

	"aimili-vpngate-go/pkg/dns"
)

var (
	primaryUDPDNSAddr   = &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 53}
	secondaryUDPDNSAddr = &net.UDPAddr{IP: net.IPv4(8, 8, 8, 8), Port: 53}
)

const (
	udpDNSFirstWait = 400 * time.Millisecond
	udpDNSMaxWait   = 1800 * time.Millisecond
)

// encodeDNSQueryA builds an RFC 1035 standard A-record DNS query for host.
func encodeDNSQueryA(txID uint16, host string) ([]byte, error) {
	trimmed := strings.TrimSuffix(strings.TrimSpace(host), ".")
	if trimmed == "" || len(trimmed) > 253 {
		return nil, fmt.Errorf("invalid hostname %q", host)
	}
	labels := strings.Split(trimmed, ".")
	wireLen := 12 + 1 + 4
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return nil, fmt.Errorf("invalid DNS label in %q", host)
		}
		wireLen += 1 + len(label)
	}

	pkt := make([]byte, wireLen)
	binary.BigEndian.PutUint16(pkt[0:2], txID)
	binary.BigEndian.PutUint16(pkt[2:4], 0x0100) // Standard query, Recursion Desired
	binary.BigEndian.PutUint16(pkt[4:6], 1)      // QDCOUNT = 1

	off := 12
	for _, label := range labels {
		pkt[off] = byte(len(label))
		off++
		copy(pkt[off:], label)
		off += len(label)
	}
	pkt[off] = 0
	off++
	binary.BigEndian.PutUint16(pkt[off:off+2], 1)   // QTYPE = A
	binary.BigEndian.PutUint16(pkt[off+2:off+4], 1) // QCLASS = IN
	return pkt, nil
}

func skipDNSName(pkt []byte, offset int) (int, error) {
	if offset < 0 || offset >= len(pkt) {
		return 0, errors.New("dns name out of bounds")
	}
	seen := 0
	for {
		if offset >= len(pkt) || seen > 128 {
			return 0, errors.New("truncated dns name")
		}
		b := pkt[offset]
		if b == 0 {
			return offset + 1, nil
		}
		switch b & 0xc0 {
		case 0xc0:
			if offset+2 > len(pkt) {
				return 0, errors.New("truncated dns pointer")
			}
			return offset + 2, nil
		case 0x00:
			offset += 1 + int(b)
			seen++
		default:
			return 0, errors.New("invalid dns label prefix")
		}
	}
}

// decodeDNSResponseA parses an RFC 1035 DNS reply and extracts IPv4 A records + bounded TTL.
func decodeDNSResponseA(pkt []byte, expectedTxID uint16) ([]net.IP, time.Duration, error) {
	if len(pkt) < 12 {
		return nil, 0, errors.New("dns packet too short")
	}
	if binary.BigEndian.Uint16(pkt[0:2]) != expectedTxID {
		return nil, 0, errors.New("dns txid mismatch")
	}
	flags := binary.BigEndian.Uint16(pkt[2:4])
	if flags&0x8000 == 0 {
		return nil, 0, errors.New("not a dns response")
	}
	if rcode := flags & 0x000f; rcode != 0 {
		return nil, 0, fmt.Errorf("dns rcode=%d", rcode)
	}

	qdCount := int(binary.BigEndian.Uint16(pkt[4:6]))
	anCount := int(binary.BigEndian.Uint16(pkt[6:8]))

	off := 12
	for i := 0; i < qdCount; i++ {
		next, err := skipDNSName(pkt, off)
		if err != nil {
			return nil, 0, err
		}
		if next+4 > len(pkt) {
			return nil, 0, errors.New("truncated dns question")
		}
		off = next + 4
	}

	var ips []net.IP
	minTTL := uint32(0)
	for i := 0; i < anCount; i++ {
		next, err := skipDNSName(pkt, off)
		if err != nil {
			return nil, 0, err
		}
		if next+10 > len(pkt) {
			return nil, 0, errors.New("truncated dns answer header")
		}
		rtype := binary.BigEndian.Uint16(pkt[next : next+2])
		rclass := binary.BigEndian.Uint16(pkt[next+2 : next+4])
		ttl := binary.BigEndian.Uint32(pkt[next+4 : next+8])
		rdLen := int(binary.BigEndian.Uint16(pkt[next+8 : next+10]))
		rdataOff := next + 10
		if rdataOff+rdLen > len(pkt) {
			return nil, 0, errors.New("truncated dns answer rdata")
		}
		if rtype == 1 && rclass == 1 && rdLen == 4 {
			ip := net.IPv4(pkt[rdataOff], pkt[rdataOff+1], pkt[rdataOff+2], pkt[rdataOff+3]).To4()
			if ip != nil {
				ips = append(ips, ip)
				if ttl > 0 && (minTTL == 0 || ttl < minTTL) {
					minTTL = ttl
				}
			}
		}
		off = rdataOff + rdLen
	}

	if len(ips) == 0 {
		return nil, 0, errors.New("no A records in dns response")
	}

	ttlDur := time.Duration(minTTL) * time.Second
	if ttlDur < 30*time.Second {
		ttlDur = 30 * time.Second
	}
	if ttlDur > dns.MaxCacheTTL {
		ttlDur = dns.MaxCacheTTL
	}
	return ips, ttlDur, nil
}

// lookupIPv4ViaUDP performs a fast 1-RTT UDP 53 query through the tunnel with hedged failover.
// If the relay drops UDP 53, it returns an error so DualEngineResolver falls back to tunnel DoH.
func (t *Tunnel) lookupIPv4ViaUDP(ctx context.Context, host string) ([]net.IP, time.Duration, error) {
	txID := uint16(rand.Uint32())
	query, err := encodeDNSQueryA(txID, host)
	if err != nil {
		return nil, 0, err
	}

	pconn, err := t.OpenPacketConn()
	if err != nil {
		return nil, 0, err
	}
	defer pconn.Close()

	stopWake := context.AfterFunc(ctx, func() {
		_ = pconn.Close()
	})
	defer stopWake()

	overallDeadline := time.Now().Add(udpDNSMaxWait)
	if d, ok := ctx.Deadline(); ok && d.Before(overallDeadline) {
		overallDeadline = d
	}

	if _, err := pconn.WriteTo(query, primaryUDPDNSAddr); err != nil {
		return nil, 0, err
	}

	buf := make([]byte, 1500)
	hedged := false
	phaseDeadline := time.Now().Add(udpDNSFirstWait)
	if phaseDeadline.After(overallDeadline) {
		phaseDeadline = overallDeadline
		hedged = true
	}

	for {
		if ctx.Err() != nil {
			return nil, 0, ctx.Err()
		}
		now := time.Now()
		if !now.Before(overallDeadline) {
			return nil, 0, ErrTimeout
		}

		readDL := overallDeadline
		if !hedged && phaseDeadline.Before(overallDeadline) {
			readDL = phaseDeadline
		}
		_ = pconn.SetReadDeadline(readDL)

		n, from, rerr := pconn.ReadFrom(buf)
		if rerr != nil {
			if ctx.Err() != nil {
				return nil, 0, ctx.Err()
			}
			if !hedged && time.Now().Before(overallDeadline) {
				hedged = true
				if _, werr := pconn.WriteTo(query, secondaryUDPDNSAddr); werr != nil {
					return nil, 0, werr
				}
				continue
			}
			return nil, 0, rerr
		}

		ufrom, ok := from.(*net.UDPAddr)
		if !ok || ufrom.Port != 53 {
			continue
		}
		if !ufrom.IP.Equal(primaryUDPDNSAddr.IP) && !ufrom.IP.Equal(secondaryUDPDNSAddr.IP) {
			continue
		}

		ips, ttl, derr := decodeDNSResponseA(buf[:n], txID)
		if derr != nil {
			if !hedged && time.Now().Before(overallDeadline) {
				hedged = true
				_, _ = pconn.WriteTo(query, secondaryUDPDNSAddr)
				continue
			}
			return nil, 0, derr
		}
		return ips, ttl, nil
	}
}
