package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"sync"

	"github.com/google/netstack/tcpip/header"

	"github.com/howeyc/crc16"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
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
			// ipPkt := header.IPv4(pkt)
			// udpPkt := header.UDP(ipPkt.Payload())
			// _, err := conn.Write(udpPkt.Payload())
			_, err := conn.Write(pkt)
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

func sendToIpv2(crc2ConnMap map[uint16]*net.UDPConn, mu sync.RWMutex, pkt []byte, addr *net.IP, port int) {
	crc, err := PktCrc16(pkt)
	if err != nil {
		log.Printf("Error calculation the crc of the incoming packet: %s\n", err)
		return
	}

	//ipPkt := header.IPv4(pkt)
	//udpPkt := header.UDP(ipPkt.Payload())
	mu.RLock()
	conn, ok := crc2ConnMap[crc]
	mu.RUnlock()
	if ok {
		_, err := conn.Write(pkt)
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

		_, err = conn.Write(pkt)
		if err != nil {
			log.Printf("Error sending message via UDP socket: %s\n", err)
			return
		}
	}
}

func sendToIpv3(conn *net.UDPConn, pkt []byte) {
	// ipPkt := header.IPv4(pkt)
	// udpPkt := header.UDP(ipPkt.Payload())
	// _, err := conn.Write(udpPkt.Payload())
	_, err := conn.Write(pkt)
	if err != nil {
		log.Printf("Error sending message via udp socket: %s\n", err)
		return
	}
}

func main() {
	len := len(os.Args)

	version, _ := strconv.Atoi(os.Args[len-1])
	switch version {
	case 1:
		main1()
	case 2:
		main2()
	case 3:
		main3()
	case 4:
		main4()
	default:
		log.Fatalf("No version selected as input: %s\n", version)
	}
}

func main1() {
	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	ninf := args[0]
	ip := net.ParseIP(args[1])
	port, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatalf("Error parsing port from argument %s: %s\n", args[2], err)
	}

	nif, err := net.InterfaceByName(ninf)
	if err != nil {
		log.Fatalf("Error accessing to interface %s\n", ninf)
	}

	eth, err := raw.ListenPacket(nif, 0x800, nil)
	if err != nil {
		log.Fatalf("Error opening ethernet interface: %s\n", err)
	}
	defer eth.Close()

	buff := make([]byte, nif.MTU)

	var eFrame ethernet.Frame

	pktChan := make(chan []byte)

	go sendToIp(&ip, port, pktChan)

	for {
		size, _, err := eth.ReadFrom(buff)
		if err != nil {
			log.Fatalf("Error reading from ethernet interface: %s\n", err)
		}

		err = (&eFrame).UnmarshalBinary(buff[:size])
		if err != nil {
			log.Fatalf("Error unmarshalling received message: %s\n", err)
		}

		ipBuff := []byte(eFrame.Payload)

		switch ipBuff[9] {
		case 17:
			// UDP
			pktChan <- ipBuff
		}
	}

}

func main2() {
	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	ninf := args[0]
	ip := net.ParseIP(args[1])
	port, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatalf("Error parsing port from argument %s: %s\n", args[2], err)
	}

	nif, err := net.InterfaceByName(ninf)
	if err != nil {
		log.Fatalf("Error accessing to interface %s\n", ninf)
	}

	eth, err := raw.ListenPacket(nif, 0x800, nil)
	if err != nil {
		log.Fatalf("Error opening ethernet interface: %s\n", err)
	}
	defer eth.Close()

	buff := make([]byte, nif.MTU)

	var eFrame ethernet.Frame

	var crc2ConnMap map[uint16]*net.UDPConn = make(map[uint16]*net.UDPConn)
	var mu sync.RWMutex

	for {
		size, _, err := eth.ReadFrom(buff)
		if err != nil {
			log.Fatalf("Error reading from ethernet interface: %s\n", err)
		}

		err = (&eFrame).UnmarshalBinary(buff[:size])
		if err != nil {
			log.Fatalf("Error unmarshalling received message: %s\n", err)
		}

		//ipBuff := []byte(eFrame.Payload)

		switch eFrame.Payload[9] {
		case 17:
			// UDP
			go sendToIpv2(crc2ConnMap, mu, eFrame.Payload, &ip, port)
		}
	}
}

func main3() {
	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	ninf := args[0]
	ip := net.ParseIP(args[1])
	port, err := strconv.Atoi(args[2])
	if err != nil {
		log.Fatalf("Error parsing port from argument %s: %s\n", args[2], err)
	}

	nif, err := net.InterfaceByName(ninf)
	if err != nil {
		log.Fatalf("Error accessing to interface %s\n", ninf)
	}

	eth, err := raw.ListenPacket(nif, 0x800, nil)
	if err != nil {
		log.Fatalf("Error opening ethernet interface: %s\n", err)
	}
	defer eth.Close()

	buff := make([]byte, nif.MTU)

	var eFrame ethernet.Frame

	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{ip, port, ""})
	if err != nil {
		log.Fatalf("Error opening UDP connection: %s\n", err)
	}

	for {
		size, _, err := eth.ReadFrom(buff)
		if err != nil {
			log.Fatalf("Error reading from ethernet interface: %s\n", err)
		}

		err = (&eFrame).UnmarshalBinary(buff[:size])
		if err != nil {
			log.Fatalf("Error unmarshalling received message: %s\n", err)
		}

		ipBuff := []byte(eFrame.Payload)

		switch ipBuff[9] {
		case 17:
			// UDP
			sendToIpv3(conn, ipBuff)
		}
	}
}

func main4() {
	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	ip := net.ParseIP(args[0])
	port, err := strconv.Atoi(args[1])
	if err != nil {
		log.Fatalf("Error parsing port from argument %s: %s\n", args[2], err)
	}

	inConn, err := net.ListenIP("ip4:udp", &net.IPAddr{net.IPv4(0, 0, 0, 0), ""})
	if err != nil {
		log.Fatalf("Error opening ip connection: %s\n", err)
	}
	defer inConn.Close()

	buff := make([]byte, 65536)

	outConn, err := net.DialUDP("udp", nil, &net.UDPAddr{ip, port, ""})
	if err != nil {
		log.Fatalf("Error opening UDP connection: %s\n", err)
	}

	log.Println("Before for loop")

	var mu sync.RWMutex
	portConnMap := make(map[int]bool)
	stopChan := make(chan struct{})

	go OpenUDPConnection(5832, mu, portConnMap, stopChan)

	for {
		size, err := inConn.Read(buff)
		if err != nil {
			log.Printf("Error reading from ip socket: %s\n", err)
			continue
		}

		ipPkt := header.IPv4(buff[:size])
		ipBuff := []byte(buff[:size])

		switch ipPkt.Protocol() {
		case 17:
			// UDP
			sendToIpv3(outConn, ipBuff)
		default:
			continue
		}
	}
}

func OpenUDPConnection(port int, mu sync.RWMutex, portConnMap map[int]bool, stopChan chan struct{}) {
	mu.RLock()
	_, ok := portConnMap[port]
	mu.RUnlock()
	if ok {
		return
	} else {
		mu.Lock()
		portConnMap[port] = true
		mu.Unlock()

		_, err := net.ListenUDP("udp", &net.UDPAddr{net.IPv4(0, 0, 0, 0), port, ""})
		if err != nil {
			log.Fatalf("Error opening connection: %s\n", err)
		}
		<-stopChan
	}
}
