package main

import (
	"faasrouter/cnt"
	"faasrouter/udp"
	"faasrouter/utils"
	"log"
	"net"
	"os"
)

var RLogger *log.Logger

func main() {

	args := os.Args[1:]
	len := len(args)
	if len < 3 {
		log.Fatalf("Error provided parameter are not enough, expected 3, provided %d\n", len)
	}

	hostname := args[0]
	auth := args[1]
	action := args[2]

	stop := make(chan struct{})
	incPktChan := make(chan []byte, 200)

	var addr *net.IPAddr = &net.IPAddr{net.ParseIP(utils.LOCAL_INTERFACE), "ip4:1"}
	go cnt.Handler(incPktChan, stop, hostname, auth, action, RLogger)
	//go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	go udp.HandleIncomingRequestsFromIPv4(addr, incPktChan, RLogger)
	<-stop
}
