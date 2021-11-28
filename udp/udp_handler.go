package udp

import (
	"context"
	"log"
	"net"
	"nfgateway/cnt"
	"nfgateway/utils"
)

const (
	MAX_UDP_PACKET_SIZE = 65535
)

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

	outChan, lock := ruleMap.GetChan(pktBuff)
	if outChan == nil {
		utils.RLogger.Println("Error no out chan available for incoming packet, packet dropped")
		return
	}

	ctx := context.TODO()
	lock.Acquire(ctx, 1)
	cnt.SendToContainer3(pktBuff, outChan)
	lock.Release(1)
	//utils.RLogger.Printf("Packet received from %s:%d headed to %s:%d sent to container\n", srcIP, src, trgIP, trg)
}
