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

	if len(args) < 2 {
		log.Fatalln("Error too few parameters!")
	}

	port, err := strconv.Atoi(args[1])
	if err != nil {
		log.Fatalln("Error converting input parameter to integer!")
	}

	lst, err := net.ListenTCP("tcp", &net.TCPAddr{net.IPv4(0, 0, 0, 0), port, ""})
	if err != nil {
		log.Fatalf("Error opening TCP connection on port %d: %s\n", port)
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
		conn, err := lst.AcceptTCP()
		log.Println("Start accepting conn")
		if err != nil {
			log.Fatalf("Error opening TCP connection: %s\n", err)
		}

		for {
			size, err := conn.Read(buff)
			if err != nil {
				log.Printf("Error reading from TCP socket: %s\n", err)
				break
			}

			b.Add(size)
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
