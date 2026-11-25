package sstp

import (
	"fmt"
	"net"

	"gvisor.dev/gvisor/pkg/tcpip"
	"gvisor.dev/gvisor/pkg/tcpip/header"
	"gvisor.dev/gvisor/pkg/tcpip/link/channel"
	"gvisor.dev/gvisor/pkg/tcpip/network/ipv4"
	"gvisor.dev/gvisor/pkg/tcpip/stack"
	"gvisor.dev/gvisor/pkg/tcpip/transport/icmp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/tcp"
	"gvisor.dev/gvisor/pkg/tcpip/transport/udp"
)

const sstpNICID = tcpip.NICID(1)

type channelNotification struct {
	ch chan struct{}
}

func (n *channelNotification) WriteNotify() {
	select {
	case n.ch <- struct{}{}:
	default:
	}
}

func newNetstack(assignedIP net.IP) (*stack.Stack, *channel.Endpoint, chan struct{}, error) {
	ip4 := assignedIP.To4()
	if ip4 == nil {
		return nil, nil, nil, fmt.Errorf("netstack requires a valid IPv4 address, got %v", assignedIP)
	}

	s := stack.New(stack.Options{
		NetworkProtocols: []stack.NetworkProtocolFactory{
			ipv4.NewProtocol,
		},
		TransportProtocols: []stack.TransportProtocolFactory{
			tcp.NewProtocol,
			udp.NewProtocol,
			icmp.NewProtocol4,
		},
	})

	sack := tcpip.TCPSACKEnabled(true)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &sack)

	moderate := tcpip.TCPModerateReceiveBufferOption(true)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &moderate)

	rcvRange := tcpip.TCPReceiveBufferSizeRangeOption{
		Min:     4 * 1024,
		Default: 256 * 1024,
		Max:     2 * 1024 * 1024,
	}
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &rcvRange)

	sndRange := tcpip.TCPSendBufferSizeRangeOption{
		Min:     4 * 1024,
		Default: 256 * 1024,
		Max:     2 * 1024 * 1024,
	}
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &sndRange)

	cc := tcpip.CongestionControlOption("cubic")
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &cc)

	delay := tcpip.TCPDelayEnabled(true)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &delay)

	recovery := tcpip.TCPRecovery(tcpip.TCPRACKLossDetection)
	_ = s.SetTransportProtocolOption(tcp.ProtocolNumber, &recovery)

	ep := channel.New(512, DefaultTunnelMTU, "")
	notifyCh := make(chan struct{}, 1)
	ep.AddNotify(&channelNotification{ch: notifyCh})

	if err := s.CreateNIC(sstpNICID, ep); err != nil {
		ep.Close()
		s.Close()
		return nil, nil, nil, fmt.Errorf("create gvisor NIC failed: %v", err)
	}

	addr := tcpip.AddrFrom4Slice(ip4)
	if err := s.AddProtocolAddress(sstpNICID, tcpip.ProtocolAddress{
		Protocol:          ipv4.ProtocolNumber,
		AddressWithPrefix: addr.WithPrefix(),
	}, stack.AddressProperties{}); err != nil {
		ep.Close()
		s.Close()
		return nil, nil, nil, fmt.Errorf("assign gvisor IP failed: %v", err)
	}

	s.SetRouteTable([]tcpip.Route{
		{
			Destination: header.IPv4EmptySubnet,
			NIC:         sstpNICID,
		},
	})

	return s, ep, notifyCh, nil
}
