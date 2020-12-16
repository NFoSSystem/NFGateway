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

	if len(args) < 1 {
		log.Fatalln("Error too few parameters!")
	}

	var receive bool = false
	if args[1] == "r" {
		receive = true
	}

	if receive {
		if len(args) < 2 {
			log.Fatalln("Error too few parameters!")
		}

		port, err := strconv.Atoi(args[2])
		if err != nil {
			log.Fatalln("Error converting input parameter to integer!")
		}

		conn, err := net.ListenUDP("udp", &net.UDPAddr{net.IPv4(0, 0, 0, 0), port, ""})
		if err != nil {
			log.Fatalf("Error opening UDP connection on port %d: %s\n", port)
		}

		bytesReceived := 0
		buff := make([]byte, 1460)
		var testRunning bool = true

		var startTime int64
		for testRunning {
			size, err := conn.Read(buff)
			if err != nil {
				log.Printf("Error reading from UDP socket: %s\n", err)
				continue
			}
			if startTime == 0 {
				startTime = time.Now().UnixNano()
				time.AfterFunc(10*time.Second, func() {
					testRunning = false
				})
			}

			bytesReceived += size
		}

		size, err := conn.Read(buff)
		if err != nil {
			log.Printf("Error reading from UDP socket: %s\n", err)
		}
		bytesReceived += size
		elapsedTime := time.Now().UnixNano() - startTime

		log.Printf("BytesReceived: %d - StartTime: %d - ElapsedTime: %d\n", bytesReceived, startTime, elapsedTime)
		log.Printf("Throughtput calculated: %.2f\n", calculateThroughput(bytesReceived, elapsedTime))
	} else {
		if len(args) < 3 {
			log.Fatalln("Error too few parameters!")
		}

		buff := make([]byte, 1460)
		buff[0] = 1
		buff[1459] = 1

		addr := net.ParseIP(args[2])
		if args[2] == "" || addr == nil {
			log.Fatalf("Error provided address %s not valid!", args[2])
		}

		port, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatalln("Error converting input parameter to integer!")
		}

		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{addr, port, ""})
		if err != nil {
			log.Fatalf("Error opening UDP socket to %s:%d: %s\n", addr, port, err)
		}
		defer conn.Close()

		timeOut := false
		startTime := time.Now().UnixNano()
		time.AfterFunc(10*time.Second, func() {
			timeOut = true
		})
		bytesSent := 0
		for !timeOut {
			size, err := conn.Write(buff)
			if err != nil {
				log.Fatalf("Error writing buff to UDP socket headed to %s:%d: %s\n", addr, port, err)
			}
			bytesSent += size

		}

		elapsedTime := time.Now().UnixNano() - startTime
		log.Printf("Throughtput calculated: %.2f\n", calculateThroughput(bytesSent, elapsedTime))
	}
}

func calculateThroughput(bytes int, elapsedTime int64) float64 {
	return float64(bytes*8) / float64(elapsedTime/1000)
}

func terminateTestAtReceiver(remoteAddr *net.IP, stopChan chan struct{}) {
	conn, err := net.DialTCP("tcp", nil, &net.TCPAddr{*remoteAddr, 5000, ""})
	if err != nil {
		log.Printf("Error opening TCP socket on port 5000: %s\n", err)
		return
	}
	defer conn.Close()

	_, err = conn.Write([]byte("End test"))
	if err != nil {
		log.Printf("Error writing to TCP socket at port 5000: %s\n", err)
		return
	}
	close(stopChan)
}
