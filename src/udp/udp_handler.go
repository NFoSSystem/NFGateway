package udp

import (
	"fmt"
	"log"
	"net"
	"utils"
)

const (
	MAX_UDP_PACKET_SIZE = 65535
	UDP_SRC_FIRST_WORD  = 20
	UDP_TRG_FIRST_WORD  = 22
)

func HandleIncomingRequestsFromIPv4(addr *net.IPAddr) {
	conn, err := net.ListenIP("ip4:udp", addr)
	if err != nil {
		fmt.Printf("Error opening UDP connection on interface %s\n", *addr)
		return
	}

	getPortFromPacket := utils.GetPortsFromBytes(UDP_SRC_FIRST_WORD, UDP_SRC_FIRST_WORD+1,
		UDP_TRG_FIRST_WORD, UDP_TRG_FIRST_WORD+1)

	bp := utils.NewBuffersPool(MAX_UDP_PACKET_SIZE, 5)
	for {
		pktBuff := bp.Next()
		if pktBuff == nil {
			fmt.Println("Error the provided buffer is nil!")
		}
		_, err = conn.Read(pktBuff.Buff())
		src, trg, _ := getPortFromPacket(pktBuff.Buff())
		defer conn.Close()
		if err != nil {
			fmt.Errorf("Error reading UDP packet from interface %s", addr.IP)
			pktBuff.Release()
			return
		}
		log.Printf("Received UDP packet from port %d headed to port %d", src, trg)

		go utils.SendToEndpoint(pktBuff, "http://127.0.0.1:8080")
	}
}
