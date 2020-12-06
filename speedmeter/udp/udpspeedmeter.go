package main

import (
	"log"
	"net"
	"os"
	"strconv"
	"sync"
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

		b := new(BitCounter)

		var kb uint32 = 7
		var mb uint32 = 17
		var gb uint32 = 27

		go func() {
			t := time.NewTicker(time.Second)
			for {
				select {
				case <-t.C:
					res := b.GetAndClean()
					var unit string
					if (res >> gb) > 0 {
						unit = "Gb/s"
						res = res >> gb
					} else if (res >> mb) > 0 {
						unit = "Mb/s"
						res = res >> mb
					} else if (res >> kb) > 0 {
						unit = "Kb/s"
						res = res >> kb
					} else {
						unit = "b/s"
					}

					log.Printf("Speed %d %s", res, unit)
				default:
					continue
				}
			}
		}()

		buff := make([]byte, 65535)
		for {
			size, err := conn.Read(buff)
			if err != nil {
				log.Printf("Error reading from TCP socket: %s\n", err)
				continue
			}

			b.Add(size)
		}
	} else {
		if len(args) < 3 {
			log.Fatalln("Error too few parameters!")
		}

		buff := make([]byte, 65507)
		buff[0] = 1
		buff[65506] = 1

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

		for {
			_, err := conn.Write(buff)
			if err != nil {
				log.Fatalf("Error writing buff to UDP socket headed to %s:%d: %s\n", addr, port, err)
			}
		}
	}
}

type BitCounter struct {
	bits uint32
	mu   sync.Mutex
}

func (b *BitCounter) Add(size int) {
	b.mu.Lock()
	b.bits += uint32(size)
	b.mu.Unlock()
}

func (b *BitCounter) GetAndClean() uint32 {
	b.mu.Lock()
	res := b.bits
	b.bits = 0
	b.mu.Unlock()
	return res
}
