package tcp

import (
	"fmt"
	"log"
	"net"
	"utils"
)

const (
	MAX_TCP_PACKET_SIZE = 65535
	TCP_SRC_FIRST_WORD  = 20
	TCP_TRG_FIRST_WORD  = 22
	LOCAL_INTERFACE     = "127.0.0.1"
)

func HandleIncomingRequestsFromIPv4(addr *net.IPAddr) {
	conn, err := net.ListenIP("ip4:tcp", addr)
	if err != nil {
		fmt.Printf("Error opening TCP connection on interface %s\n", addr.IP)
		return
	}

	getPortFromPacket := utils.GetPortsFromBytes(TCP_SRC_FIRST_WORD, TCP_SRC_FIRST_WORD+1,
		TCP_TRG_FIRST_WORD, TCP_TRG_FIRST_WORD+1)

	bp := utils.NewBuffersPool(MAX_TCP_PACKET_SIZE, 5)
	for {
		pktBuff := bp.Next()
		if pktBuff == nil {
			fmt.Println("Error the provided buffer is nil!")
		}
		_, err = conn.Read(pktBuff.Buff())
		_, trg, _ := getPortFromPacket(pktBuff.Buff())
		defer conn.Close()
		if err != nil {
			fmt.Errorf("Error reading TCP packet from interface %s", addr.IP)
			pktBuff.Release()
			return
		}
		log.Printf("Received TCP packet from port %d", trg)

		go utils.SendToEndpoint(pktBuff, "http://127.0.0.1:8080")
	}
}
