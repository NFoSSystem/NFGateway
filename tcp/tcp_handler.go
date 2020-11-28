package tcp

import (
	"fmt"
	"log"
	"net"
	"utils"

	"github.com/howeyc/crc16"
)

const (
	MAX_TCP_PACKET_SIZE   = 65535
	TCP_IP_SRC_FIRST_BYTE = 12
	TCP_IP_TRG_FIRST_BYTE = 16
	TCP_SRC_FIRST_WORD    = 20
	TCP_TRG_FIRST_WORD    = 22
	LOCAL_INTERFACE       = "127.0.0.1"
)

var GetIPsFromPkt func([]byte) (net.IP, net.IP, error) = utils.GetIPsFromBytes(TCP_IP_SRC_FIRST_BYTE, TCP_IP_TRG_FIRST_BYTE)

var GetPortsFromPkt func([]byte) (uint16, uint16, error) = utils.GetPortsFromBytes(TCP_SRC_FIRST_WORD, TCP_SRC_FIRST_WORD+1,
	TCP_TRG_FIRST_WORD, TCP_TRG_FIRST_WORD+1)

/*
PktCrc16 calculates Crc16 from source IP, target IP, source port and target port from provided packet.
*/
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

func HandleIncomingRequestsFromIPv4(addr *net.IPAddr, pktChan chan []byte, logger *log.Logger) {
	conn, err := net.ListenIP("ip4:tcp", addr)
	if err != nil {
		fmt.Printf("Error opening TCP connection on interface %s\n", addr.IP)
		return
	}
	defer conn.Close()

	bp := utils.NewBuffersPool(MAX_TCP_PACKET_SIZE, 5)
	for {
		pktBuff := bp.Next()
		if pktBuff == nil {
			fmt.Println("Error the provided buffer is nil!")
		}

		_, err = conn.Read(pktBuff.Buff())
		if err != nil {
			fmt.Errorf("Error reading TCP packet from interface %s", addr.IP)
			pktBuff.Release()
			return
		}

		pktChan <- []byte(pktBuff.Buff())
		pktBuff.Release()

		src, trg, _ := GetPortsFromPkt(pktBuff.Buff())
		srcIP, trgIP, _ := GetIPsFromPkt(pktBuff.Buff())

		log.Printf("Received TCP packet from %s:%d headed to %s:%d", srcIP, src, trgIP, trg)
	}
}
