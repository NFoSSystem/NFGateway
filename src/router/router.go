package main

import (
	"cnt"
	"os"
	"tcp"
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

func main() {

	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	hostname := args[0]
	auth := args[1]
	action := args[2]

	stop := make(chan struct{})
	incPktChan := make(chan []byte)
	var addr *net.IPAddr = &net.IPAddr{net.ParseIP(LOCAL_INTERFACE), "ip4:1"}
	go cnt.Handler(incPktChan, stop, hostname, auth, action)
	go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan)
	go udp.HandleIncomingRequestsFromIPv4(addr, incPktChan)
	<-stop
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

	log.Printf("len(data[header.Len:header.TotalLen]): %d", len(data[header.Len:header.TotalLen]))

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
