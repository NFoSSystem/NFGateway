package main

import (
	"faasrouter/cnt"
	"faasrouter/udp"
	"faasrouter/utils"
	"log"
	"net"
	"os"
)

const LOCAL_INTERFACE = "127.0.0.1"

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

	rpsw, err := utils.NewRedisPubSubWriter("logChan", LOCAL_INTERFACE, 6379)
	if err != nil {
		log.Println("Error creating RedisPubSubWriter: %s", rpsw)
	}

	rLogger := log.New(rpsw, "[Gateway]", log.Ldate|log.Lmicroseconds)

	var addr *net.IPAddr = &net.IPAddr{net.ParseIP(LOCAL_INTERFACE), "ip4:1"}
	go cnt.Handler(incPktChan, stop, hostname, auth, action, rLogger)
	//go tcp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	go udp.HandleIncomingRequestsFromIPv4(addr, incPktChan, rLogger)
	<-stop
}
