package sstp

import (
	"errors"
	"time"
)

var (
	ErrTunnelClosed    = errors.New("sstp tunnel is closed")
	ErrHandshakeFailed = errors.New("sstp handshake failed")
	ErrAuthFailed      = errors.New("sstp pap authentication failed")
	ErrIPCPFailed      = errors.New("sstp ipcp negotiation failed")
	ErrTimeout         = errors.New("sstp operation timed out")
	ErrTunnelStale     = errors.New("sstp tunnel peer stopped responding")
)

const (
	DefaultConnectTimeout = 15 * time.Second
	DefaultReadTimeout    = 30 * time.Second
	DefaultWriteTimeout   = 10 * time.Second
	DefaultEchoInterval   = 20 * time.Second
	DefaultTCPMSS         = 1360

	// DefaultIdleTimeout bounds how long a tunnel may stay completely silent
	// before it is treated as dead. The keepalive loop sends an SSTP echo every
	// DefaultEchoInterval and SoftEther relays also emit their own keepalives, so
	// several consecutive silent rounds mean the session is gone even though the
	// underlying TCP socket has not reported an error yet. Without this bound a
	// silently black-holed relay keeps reporting "connected" for many minutes
	// while every proxied request fails.
	DefaultIdleTimeout = 75 * time.Second
)

// SSTP Header Constants
const (
	SSTPVersion = 0x10 // Major 1, Minor 0

	SSTPFlagData    = 0x00
	SSTPFlagControl = 0x01

	// SSTP Control Message Types
	MsgCallConnectRequest = 0x0001
	MsgCallConnectAck     = 0x0002
	MsgCallConnectNak     = 0x0003
	MsgCallConnected      = 0x0004
	MsgCallAbort          = 0x0005
	MsgCallDisconnect     = 0x0006
	MsgCallDisconnectAck  = 0x0007
	MsgEchoRequest        = 0x0008
	MsgEchoResponse       = 0x0009

	// SSTP Attribute IDs
	AttrEncapsulatedProtocolID = 0x01
	AttrStatusInfo             = 0x02
	AttrCryptoBinding          = 0x03
	AttrCryptoBindingReq       = 0x04

	// Encapsulated Protocol IDs
	EncapsulatedProtoPPP = 0x0001
)

// PPP Protocols
const (
	ProtoLCP  uint16 = 0xc021
	ProtoPAP  uint16 = 0xc023
	ProtoCHAP uint16 = 0xc223
	ProtoIPCP uint16 = 0x8021
	ProtoIPv4 uint16 = 0x0021
	ProtoIPv6 uint16 = 0x0057
)

// PPP Codes
const (
	PPPCodeConfigureRequest = 1
	PPPCodeConfigureAck     = 2
	PPPCodeConfigureNak     = 3
	PPPCodeConfigureReject  = 4
	PPPCodeTerminateRequest = 5
	PPPCodeTerminateAck     = 6
	PPPCodeCodeReject       = 7
	PPPCodeEchoRequest      = 9
	PPPCodeEchoReply        = 10
)

// LCP Options
const (
	LCPOptMRU  = 1
	LCPOptAuth = 3
)

// PAP Codes
const (
	PAPCodeAuthRequest = 1
	PAPCodeAuthAck     = 2
	PAPCodeAuthNak     = 3
)

// IPCP Options
const (
	IPCPOptIPAddress = 3
	IPCPOptPrimaryDNS   = 129
	IPCPOptSecondaryDNS = 131
)

// TCP Flags
const (
	TCPFlagFIN = 0x01
	TCPFlagSYN = 0x02
	TCPFlagRST = 0x04
	TCPFlagPSH = 0x08
	TCPFlagACK = 0x10
	TCPFlagURG = 0x20
)
