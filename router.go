package main

import (
	"faasrouter/cnt"
	"faasrouter/initnf"
	"faasrouter/nat"
	"faasrouter/udp"
	"faasrouter/utils"
	"log"
	"net"
	"os"
	"regexp"
	"runtime/debug"
	"strconv"
	"time"

	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
)

func main() {

	args := os.Args[1:]
	len := len(args)
	if len < 4 {
		log.Fatalf("Error provided parameter are not enough, expected 4, provided %d\n", len)
	}

	hostname := args[0]
	auth := args[1]
	redisIp, redisPort := parseRedisAddressParam(args[2])
	ninf := args[3:]

	go func() {
		tick := time.NewTicker(5 * time.Second)
		for {
			<-tick.C
			debug.FreeOSMemory()
		}
	}()

	stop := make(chan struct{})

	log.Printf("Starting faasrouter at %d", time.Now().UnixNano())

	initnf.InitNat()
	initnf.InitDhcp()

	rl := cnt.InitRuleMap(stop, hostname, auth, utils.RLogger, redisIp, redisPort)
	//go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	go nat.ListenForMappingRequests(utils.CPMap, utils.PCMap)

	for _, ni := range ninf {
		go readFromNetworkInterface(ni, rl, utils.RLogger)
	}
	<-stop
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
			if buff[23] == 67 {
				go udp.HandleIncomingRequestsFromIPv4(buff, rl, RLogger)
			} else {
				go udp.HandleIncomingRequestsFromIPv4(ipBuff, rl, RLogger)
			}
		}
	}
}

func parseRedisAddressParam(redisAddress string) (string, int) {
	rex := regexp.MustCompile("(.*):(\\d+)")
	matches := rex.FindStringSubmatch(redisAddress)
	if matches == nil || len(matches) < 3 {
		utils.RLogger.Fatalf("Error parsing redis address provided as paramter: %s\n", redisAddress)
	}

	port, err := strconv.Atoi(matches[2])
	if err != nil {
		utils.RLogger.Fatalf("Error parsing redis address provided as paramter: %s\n", redisAddress)
	}

	return matches[1], port
}
