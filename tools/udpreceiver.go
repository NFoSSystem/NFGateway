package main

import (
	"log"
	"net"

	"github.com/google/netstack/tcpip/header"
)

func main() {
	conn, err := net.ListenIP("ip4:udp", &net.IPAddr{net.IPv4(0, 0, 0, 0), ""})
	if err != nil {
		log.Fatalf("Error opening IP connection: %s\n", err)
	}

	buff := make([]byte, 65535)
	for {
		size, err := conn.Read(buff)
		if err != nil {
			log.Printf("Error reading from IP socket: %s\n", err)
			continue
		}

		ipPkt := header.IPv4(buff[:size])
		udpPkt := header.UDP(ipPkt)
		sourcePort := udpPkt.SourcePort()
		destPort := udpPkt.DestinationPort()
		log.Printf("Received packet from %s:%d headed to %d\n", ipPkt.SourceAddress(), sourcePort, destPort)
	}
}
