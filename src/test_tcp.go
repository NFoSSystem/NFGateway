package main

import (
	"log"
	"net"
)

func main() {
	listn, err := net.ListenTCP("tcp", &net.TCPAddr{net.IPv4(127, 0, 0, 1), 5000, ""})
	if err != nil {
		log.Fatal("Error opening TCP listner on port 5000!")
	}

	listn.Accept
	tcpConn, err := listn.AcceptTCP()
	if err != nil {
		log.Fatal("Error accepting TCP connection on port 5000")
	}

	buff := make([]byte, 65535)
	_, err = tcpConn.Read(buff)
	if err != nil {
		log.Fatal("Error reading TCP packet on port 5000")
	}

	log.Printf("Dump TCP packet: %v", buff)
}
