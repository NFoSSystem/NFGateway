package utils

import "net"

const (
	UDP_IP_SRC_FIRST_BYTE = 12
	UDP_IP_TRG_FIRST_BYTE = 16
	UDP_SRC_FIRST_WORD    = 20
	UDP_TRG_FIRST_WORD    = 22
)

var GetIPsFromPkt func([]byte) (net.IP, net.IP, error) = GetIPsFromBytes(UDP_IP_SRC_FIRST_BYTE, UDP_IP_TRG_FIRST_BYTE)

var GetPortsFromPkt func([]byte) (uint16, uint16, error) = GetPortsFromBytes(UDP_SRC_FIRST_WORD, UDP_SRC_FIRST_WORD+1,
	UDP_TRG_FIRST_WORD, UDP_TRG_FIRST_WORD+1)
