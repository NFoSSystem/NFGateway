package main

import (
	"faasrouter/cnt"
	"faasrouter/initnf"
	"faasrouter/udp"
	"faasrouter/utils"
	"log"
	"net"
	"os"

	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

var RLogger *log.Logger

func main() {

	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 4, provided %d\n", len)
	}

	ninf := args[0]
	hostname := args[1]
	auth := args[2]

	stop := make(chan struct{})

	initnf.InitNat()
	initnf.InitDhcp()

	rl := cnt.InitRuleMap(stop, hostname, auth, RLogger)
	//go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	readFromNetworkInterface(ninf, rl, RLogger)
}

func readFromNetworkInterface(interfaceName string, rl *utils.RuleMap, RLogger *log.Logger) {
	nif, err := net.InterfaceByName(interfaceName)
	if err != nil {
		RLogger.Fatalf("Error accessing to interface %s\n", interfaceName)
	}

	eth, err := raw.ListenPacket(nif, 0x800, nil)
	if err != nil {
		RLogger.Fatalf("Error opening ethernet interface: %s\n", err)
	}
	defer eth.Close()

	buff := make([]byte, nif.MTU)

	var eFrame ethernet.Frame

	for {
		size, _, err := eth.ReadFrom(buff)
		if err != nil {
			RLogger.Fatalf("Error reading from ethernet interface: %s\n", err)
		}

		err = (&eFrame).UnmarshalBinary(buff[:size])
		if err != nil {
			RLogger.Fatalf("Error unmarshalling received message: %s\n", err)
		}

		ipBuff := []byte(eFrame.Payload)

		switch ipBuff[9] {
		case 17:
			// UDP
			go udp.HandleIncomingRequestsFromIPv4(ipBuff, rl, RLogger)
		}
	}
}
