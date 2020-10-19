package main

import (
	"udp"
	"unsafe"

	//"fmt"

	"fmt"
	"log"
	"net"

	//"os"

	"golang.org/x/net/ipv4"
)

const LOCAL_INTERFACE = "127.0.0.1"

/*func main() {

	ipMaxSize := utils.FastPow(2, 16) - 1
	var addr *net.IPAddr = &net.IPAddr{net.ParseIP(LOCAL_INTERFACE), "ip4:1"}
	ltn, err := net.ListenIP("ip4:tcp", addr)
	if err != nil {
		fmt.Printf("Error opening listner on IP address %s\n", LOCAL_INTERFACE)
		os.Exit(1)
	}

	buff := make([]byte, ipMaxSize)
	log.Println("DEBUG point 1")
	_, err = ltn.Read(buff)
	if err != nil {
		fmt.Printf("Error reading input from socket!\n")
		os.Exit(1)
	}

	log.Println("DEBUG point 2")
	header, err := ipv4.ParseHeader(buff[:])
	if err != nil {
		fmt.Printf("Error parsing IP datagram header!\n")
		os.Exit(1)
	}

	fmt.Printf("Buffer content: %s\n", buff)
	fmt.Printf("Header version: %d\n", header.Version)
	fmt.Printf("Header length: %d\n", header.Len)
	fmt.Printf("Package total length: %d\n", header.TotalLen)

	parseAndSend(buff)
}*/

func main() {
	var addr *net.IPAddr = &net.IPAddr{net.ParseIP(LOCAL_INTERFACE), "ip4:1"}
	//tcp.HandleIncomingRequestsFromIPv4(addr)
	udp.HandleIncomingRequestsFromIPv4(addr)
}

func readLeadingPart(conn *net.IPConn, totalLen int, headerLen int) ([]byte, error) {
	buffLen := totalLen - headerLen
	if buffLen == 0 {
		return nil, nil
	}

	buff := make([]byte, buffLen)
	_, err := conn.Read(buff)
	if err != nil {
		fmt.Printf("Error reading packet leading payload!")
	}

	return buff, nil
}

func parseAndSend(data []byte /*, flowInterfaceMap map[string]*net.IPAddr*/) error {
	header, err := ipv4.ParseHeader(data)
	if err != nil {
		log.Fatal("Error parsing ip header!")
	}

	log.Printf("IP Header size: %d", unsafe.Sizeof(header))

	//payload := data[header.Len:header.TotalLen]

	log.Printf("len(data[header.Len:header.TotalLen]): %d", len(data[header.Len:header.TotalLen]))

	// log.Printf("Full datagram received: %v", data)

	sourcePrt, targetPrt, err := getPortFromTCP(data[header.Len:header.TotalLen])

	log.Printf("Source port: %d", sourcePrt)
	log.Printf("Target port: %d", targetPrt)

	return nil
}

func getPortFromUDP(packet []byte) (uint16, uint16, error) {
	if len(packet) < 4 {
		return 0, 0, fmt.Errorf("Error provided packet with lenght lower than 4")
	}

	log.Printf("packet[0]: %d -> %b\n", packet[0], packet[0])
	log.Printf("packet[1]: %d -> %b\n", packet[1], packet[1])

	log.Printf("packet[0]: %d -> %b\n", packet[2], packet[2])
	log.Printf("packet[1]: %d -> %b\n", packet[3], packet[3])

	source := (uint16(packet[28]|0) << 8) | uint16(packet[29])
	target := (uint16(packet[30]|0) << 8) | uint16(packet[31])

	log.Printf("(uint16(packet[0]|0) << 4) | uint16(packet[1]): %d -> %b\n", source, source)

	log.Printf("Payload: %v", packet)

	return source, target, nil
}

func getPortFromTCP(packet []byte) (uint16, uint16, error) {
	if len(packet) < 23 {
		return 0, 0, fmt.Errorf("Error provided packet with lenght lower than 4")
	}

	source := (uint16(packet[0]|0) << 8) | uint16(packet[1])
	target := (uint16(packet[2]|0) << 8) | uint16(packet[3])

	log.Printf("Payload: %v", packet)

	return source, target, nil
}
