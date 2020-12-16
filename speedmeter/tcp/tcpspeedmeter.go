package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"time"
)

func main() {
	args := os.Args

	if len(args) < 2 {
		log.Fatalln("Error too few parameters!")
	}

	var receive bool
	if args[1] == "r" {
		receive = true
	}

	if receive {
		port, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalln("Error converting input parameter to integer!")
		}

		lst, err := net.ListenTCP("tcp", &net.TCPAddr{net.IPv4(0, 0, 0, 0), port, ""})
		if err != nil {
			log.Fatalf("Error opening TCP connection on port %d: %s\n", port)
		}

		buff := make([]byte, 65535)
		for {
			conn, err := lst.AcceptTCP()
			log.Println("Start accepting conn")
			if err != nil {
				log.Fatalf("Error opening TCP connection: %s\n", err)
			}

			for {
				_, err := conn.Read(buff)
				if err != nil {
					log.Printf("Error reading from TCP socket: %s\n", err)
					break
				}
			}
		}
	} else {
		if len(args) < 3 {
			log.Fatalln("Error too few parameters!")
		}

		buff := make([]byte, 16384)
		buff[0] = 1
		buff[16383] = 1

		addr := net.ParseIP(args[2])
		if args[2] == "" || addr == nil {
			log.Fatalf("Error provided address %s not valid!", args[2])
		}

		port, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatalln("Error converting input parameter to integer!")
		}

		conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{addr, port, ""})
		if err != nil {
			log.Printf("Error opening TCP connection on port %d: %s\n", port, err)
		}

		timeOut := false
		time.AfterFunc(10*time.Second, func() {
			timeOut = true
		})

		startTime := time.Now().UnixNano()
		bytesSent := 0
		for !timeOut {
			size, err := conn.Write(buff)
			if err != nil {
				log.Printf("Error writing to TCP socket %d: %s\n", port, err)
			}

			bytesSent += size
		}

		elapsedTime := time.Now().UnixNano() - startTime
		bandwidth := float64(bytesSent*8*1000) / float64(elapsedTime)

		log.Printf("Throughput calculated: %.2f Mb\\s\n", bandwidth)
	}

}
