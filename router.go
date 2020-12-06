package main

import (
	"faasrouter/cnt"
	"faasrouter/initnf"
	"faasrouter/udp"
	"log"
	"net"
	"os"
)

var RLogger *log.Logger

func main() {

	args := os.Args[1:]
	len := len(args)
	if len < 2 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	hostname := args[0]
	auth := args[1]

	stop := make(chan struct{})

	initnf.InitNat()
	initnf.InitDhcp()

	var addr *net.IPAddr = &net.IPAddr{net.IPv4(0, 0, 0, 0), "ip4:1"}

	rl := cnt.InitRuleMap(stop, hostname, auth, RLogger)
	//go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	go udp.HandleIncomingRequestsFromIPv4(addr, rl, RLogger)
	<-stop
}
