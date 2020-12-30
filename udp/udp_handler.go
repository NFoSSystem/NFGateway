package udp

import (
	"faasrouter/utils"
	"fmt"
	"log"
	"net"

	"github.com/howeyc/crc16"
)

const (
	MAX_UDP_PACKET_SIZE = 65535
)

/*
PktCrc16 calculates Crc16 from source IP, target IP, source port and target port from provided packet.
*/
func PktCrc16(buff []byte) (uint16, error) {
	bSlice := make([]byte, 12)

	srcIP, trgIP, err := utils.GetIPsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading IP addresses from packet!")
	}

	src, trg, err := utils.GetPortsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading source and target addresses from packet!")
	}

	bSlice = append(bSlice, []byte(srcIP)...)
	bSlice = append(bSlice, []byte(trgIP)...)
	bSlice = append(bSlice, uint8(src>>8), uint8(src&255))
	bSlice = append(bSlice, uint8(trg>>8), uint8(trg&255))

	return crc16.ChecksumIBM(bSlice), nil
}

func OpenSocketOnPort() {
	stopChan := make(chan struct{})

	conn, err := net.ListenUDP("udp", &net.UDPAddr{net.IPv4(0, 0, 0, 0), 5000, ""})
	if err != nil {
		utils.RLogger.Fatalln("Error opening UDP socket on port 5000")
	}
	defer conn.Close()

	<-stopChan
}

// func HandleIncomingRequestsFromIPv4(addr *net.IPAddr, ruleMap *utils.RuleMap, logger *log.Logger) {
// 	go OpenSocketOnPort()
// 	conn, err := net.ListenIP("ip4:udp", addr)
// 	if err != nil {
// 		utils.RLogger.Fatalf("Error opening UDP connection on interface %s\n", *addr)
// 		return
// 	}
// 	defer conn.Close()

// 	//bp := utils.NewBuffersPool(MAX_UDP_PACKET_SIZE, 5)
// 	for {
// 		// pktBuff := bp.Next()
// 		// if pktBuff == nil {
// 		// 	fmt.Println("Error the provided buffer is nil!")
// 		// }

// 		pktBuff := make([]byte, 65535)

// 		size, err := conn.Read(pktBuff)
// 		if err != nil {
// 			fmt.Errorf("Error reading UDP packet from interface %s", addr.IP)
// 			//pktBuff.Release()
// 			return
// 		}

// 		pktBuff = pktBuff[:size]

// 		src, trg, _ := utils.GetPortsFromPkt(pktBuff)
// 		srcIP, trgIP, _ := utils.GetIPsFromPkt(pktBuff)

// 		if trg != 67 && trg != 5000 {
// 			continue
// 		}

// 		buff := []byte(pktBuff)

// 		outChan := ruleMap.GetChan(buff)
// 		if outChan == nil {
// 			utils.RLogger.Println("Error no out chan available for incoming packet, packet dropped")
// 			continue
// 		}

// 		outChan <- buff

// 		//pktBuff.Release()
// 		utils.RLogger.Printf("Packet received from %s:%d headed to %s:%d sent to container\n", srcIP, src, trgIP, trg)
// 	}
// }

func HandleIncomingRequestsFromIPv4(pktBuff []byte, ruleMap *utils.RuleMap, logger *log.Logger) {
	// src, trg, _ := utils.GetPortsFromPkt(pktBuff)
	// srcIP, trgIP, _ := utils.GetIPsFromPkt(pktBuff)

	// if trg != 67 && trg != 5000 {
	// 	return
	// }

	outChan := ruleMap.GetChan(pktBuff)
	if outChan == nil {
		utils.RLogger.Println("Error no out chan available for incoming packet, packet dropped")
		return
	}

	outChan <- pktBuff
	//utils.RLogger.Printf("Packet received from %s:%d headed to %s:%d sent to container\n", srcIP, src, trgIP, trg)
}
