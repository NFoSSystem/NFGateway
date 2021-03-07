package main

import (
	"errors"
	"log"
	"net"
	"os"
	"regexp"
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
		defer conn.Close()

		bytesReceived := 0
		buff := make([]byte, 1100)
		var testRunning bool = true
		var pktNum int = 0

		var startTime int64 = 0
		var endTime int64 = 0
		for testRunning {
			size, err := conn.Read(buff)
			if err != nil {
				if isDeadlineExceeded(err) {
					goto result
				}

				log.Printf("Error reading from UDP socket: %s\n", err)
				continue
			}

			if startTime == 0 {
				startTime = time.Now().UnixNano()
				time.AfterFunc(10*time.Second, func() {
					endTime = time.Now().UnixNano()
					testRunning = false
				})
				conn.SetReadDeadline(time.Now().Add(10 * time.Second))
			}

			pktNum++
			bytesReceived += size
		}

	result:
		elapsedTime := time.Now().UnixNano() - startTime
		log.Printf("Packets received: %d - Bytes received: %d - Throughtput calculated: %.2f\n", pktNum,
			bytesReceived, calculateThroughput(bytesReceived, elapsedTime))
	} else {
		if len(args) < 3 {
			log.Fatalln("Error too few parameters!")
		}

		buff := make([]byte, 1100)
		buff[0] = 1
		buff[1023] = 1

		addr := net.ParseIP(args[2])
		if args[2] == "" || addr == nil {
			log.Fatalf("Error provided address %s not valid!", args[2])
		}

		port, err := strconv.Atoi(args[3])
		if err != nil {
			log.Fatalln("Error converting input parameter to integer!")
		}

		var sleep time.Duration
		if len(args) > 4 {
			sleep = ParseSleepInputParam(args[4])
		}

		conn, err := net.DialUDP("udp", nil, &net.UDPAddr{addr, port, ""})
		if err != nil {
			log.Fatalf("Error opening UDP socket to %s:%d: %s\n", addr, port, err)
		}
		defer conn.Close()

		timeOut := false
		messagesSent := 0
		bytesSent := 0
		startTime := time.Now().UnixNano()

		time.AfterFunc(10*time.Second, func() {
			timeOut = true
		})
		for !timeOut {
			//time.Sleep(10 * time.Nanosecond)
			//time.Sleep(250 * time.Nanosecond)
			//time.Sleep(400 * time.Nanosecond)
			//time.Sleep(400 * time.Nanosecond)
			time.Sleep(sleep)
			size, err := conn.Write(buff)
			if err != nil {
				log.Fatalf("Error writing buff to UDP socket headed to %s:%d: %s\n", addr, port, err)
			}
			bytesSent += size
			messagesSent++
		}

		elapsedTime := time.Now().UnixNano() - startTime
		log.Printf("Packets sent: %d - Bytes sent: %d - Throughtput calculated: %.2f\n",
			messagesSent, bytesSent, calculateThroughput(bytesSent, elapsedTime))
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

// isDeadlineExceeded reports whether err is or wraps os.ErrDeadlineExceeded.
// We also check that the error implements net.Error, and that the
// Timeout method returns true.
func isDeadlineExceeded(err error) bool {
	nerr, ok := err.(net.Error)
	if !ok {
		return false
	}
	if !nerr.Timeout() {
		return false
	}
	if !errors.Is(err, os.ErrDeadlineExceeded) {
		return false
	}
	return true
}

func ParseSleepInputParam(sleepParam string) time.Duration {
	rgx := regexp.MustCompile("(\\d+)([num]s)")
	matchRes := rgx.FindStringSubmatch(sleepParam)
	if matchRes == nil || len(matchRes) != 3 {
		return 0
	} else {
		interval, err := strconv.Atoi(matchRes[1])
		if err != nil {
			log.Fatalf("Error parsing optional sleep param %s: %s", sleepParam, err)
		}
		return time.Duration(interval) * getDurationFromParam(matchRes[2])
	}
}

func getDurationFromParam(unit string) time.Duration {
	switch unit {
	case "ns":
		return time.Nanosecond
	case "us":
		return time.Microsecond
	case "ms":
		return time.Millisecond
	default:
		return time.Duration(0)
	}
}
