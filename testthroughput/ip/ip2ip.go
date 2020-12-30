package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/howeyc/crc16"

	"github.com/google/netstack/tcpip/header"
)

const (
	UDP_IP_SRC_FIRST_BYTE = 12
	UDP_IP_TRG_FIRST_BYTE = 16
	UDP_SRC_FIRST_WORD    = 20
	UDP_TRG_FIRST_WORD    = 22
)

func sendToIp(addr *net.IP, port int, pktChan chan []byte) {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*addr, port, ""})
	if err != nil {
		log.Fatalf("Error opening UDP connection: %s\n", err)
	}

	for {
		select {
		case pkt := <-pktChan:
			ipPkt := header.IPv4(pkt)
			udpPkt := header.UDP(ipPkt.Payload())
			_, err := conn.Write(udpPkt.Payload())
			if err != nil {
				log.Printf("Error sending message via udp socket: %s\n", err)
				continue
			}
		}
	}
}

func GetPortsFromBytes(ptr1, ptr2, ptr3, ptr4 int) func(packet []byte) (uint16, uint16, error) {
	return func(packet []byte) (uint16, uint16, error) {
		if len(packet) < 4 {
			return 0, 0, fmt.Errorf("Error provided byte slice with lenght lower than 4")
		}

		var source uint16 = (uint16(packet[ptr1]|0) << 8) | uint16(packet[ptr2])
		var target uint16 = (uint16(packet[ptr3]|0) << 8) | uint16(packet[ptr4])

		return source, target, nil
	}
}

func max(v1, v2 int) int {
	if v1 > v2 {
		return v1
	} else {
		return v2
	}
}

func GetIPsFromBytes(ptr1, ptr2 int) func([]byte) (net.IP, net.IP, error) {
	return func(pkt []byte) (net.IP, net.IP, error) {
		if len(pkt) < max(ptr1+3, ptr2+3) {
			return nil, nil, fmt.Errorf("Error byte buffer provided smaller than expected!")
		}
		return net.IPv4(pkt[ptr1], pkt[ptr1+1], pkt[ptr1+2], pkt[ptr1+3]),
			net.IPv4(pkt[ptr2], pkt[ptr2+1], pkt[ptr2+2], pkt[ptr2+3]),
			nil
	}
}

var GetIPsFromPkt func([]byte) (net.IP, net.IP, error) = GetIPsFromBytes(UDP_IP_SRC_FIRST_BYTE, UDP_IP_TRG_FIRST_BYTE)

var GetPortsFromPkt func([]byte) (uint16, uint16, error) = GetPortsFromBytes(UDP_SRC_FIRST_WORD, UDP_SRC_FIRST_WORD+1,
	UDP_TRG_FIRST_WORD, UDP_TRG_FIRST_WORD+1)

func PktCrc16(buff []byte) (uint16, error) {
	bSlice := make([]byte, 12)

	srcIP, trgIP, err := GetIPsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading IP addresses from packet!")
	}

	src, trg, err := GetPortsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading source and target addresses from packet!")
	}

	bSlice = append(bSlice, []byte(srcIP)...)
	bSlice = append(bSlice, []byte(trgIP)...)
	bSlice = append(bSlice, uint8(src>>8), uint8(src&255))
	bSlice = append(bSlice, uint8(trg>>8), uint8(trg&255))

	return crc16.ChecksumIBM(bSlice), nil
}

func main() {
	args := os.Args[1:]
	len := len(args)
	if len < 2 {
		log.Fatalf("Error provided parameter are not enough, expected 2, provided %d\n", len)
	}

	ip := net.ParseIP(args[0])
	port, err := strconv.Atoi(args[1])
	if err != nil {
		log.Fatalf("Error parsing port from argument %s: %s\n", args[2], err)
	}

	conn, err := net.ListenUDP("udp", &net.UDPAddr{net.IPv4(0, 0, 0, 0), 9286, ""})
	if err != nil {
		log.Fatalf("Error opening UDP connection: %s\n", err)
	}
	defer conn.Close()

	buff := make([]byte, 65536)

	var crc2ConnMap map[uint16]*net.UDPConn = make(map[uint16]*net.UDPConn)
	var mu sync.RWMutex

	for {
		size, err := conn.Read(buff)
		if err != nil {
			log.Fatalf("Error reading from UDP socket: %s\n", err)
		}

		go sendToIpv2(crc2ConnMap, mu, buff[:size], &ip, port)
	}

}

func sendToIpv2(crc2ConnMap map[uint16]*net.UDPConn, mu sync.RWMutex, pkt []byte, addr *net.IP, port int) {
	crc, err := PktCrc16(pkt)
	ipPkt := header.IPv4(pkt)
	udpPkt := header.UDP(ipPkt.Payload())
	if err != nil {
		log.Printf("Error calculation the crc of the incoming packet: %s\n", err)
		return
	}
	mu.RLock()
	conn, ok := crc2ConnMap[crc]
	mu.RUnlock()
	if ok {
		_, err := conn.Write(udpPkt.Payload())
		if err != nil {
			log.Printf("Error sending message via UDP socket: %s\n", err)
			return
		}
	} else {
		mu.Lock()
		conn, ok := crc2ConnMap[crc]
		if !ok {
			conn, err = net.DialUDP("udp", nil, &net.UDPAddr{*addr, port, ""})
			if err != nil {
				log.Fatalf("Error opening UDP connection: %s\n", err)
			}
			crc2ConnMap[crc] = conn
		}
		mu.Unlock()

		_, err = conn.Write(udpPkt.Payload())
		if err != nil {
			log.Printf("Error sending message via UDP socket: %s\n", err)
			return
		}
	}
}
