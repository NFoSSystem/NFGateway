package cnt

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	op "openwhisk"
	"strconv"
	"strings"
	"time"
	"udp"
	"utils"
)

const (
	CONTAINER_HANDLER_PORT            = 9645
	MAX_SENDER                        = 10
	MAX_ACTION_RUNTIME_IN_SECONDS     = 60
	ACTION_TRIGGER_TIMEOUT_IN_SECONDS = 30
)

type msg struct {
	Addr [4]uint8
	Port uint16
}

func NewMsg(addr *net.IP, port uint16) *msg {
	res := new(msg)
	split := strings.Split(addr.String(), ".")
	if split == nil || len(split) != 4 {
		res.Addr = [4]uint8{}
	} else {
		for i, frag := range split {
			byteInWord, err := strconv.ParseUint(frag, 10, 8)
			if err != nil {
				res.Addr = [4]uint8{}
				break
			} else {
				res.Addr[i] = uint8(byteInWord)
			}
		}

		res.Port = port
		fmt.Printf("Msg: %v\n", res)
	}
	return res
}

func getMsgFromBytes(bSlice []byte) *msg {
	buff := &bytes.Buffer{}
	res := msg{}
	buff.Grow(len(bSlice))
	_, err := buff.Write(bSlice)
	if err != nil {
		panic(err)
	}
	err = binary.Read(buff, binary.BigEndian, &res)
	if err != nil {
		panic(err)
	}
	return &res
}

func getBytesFromMsg(src msg) []byte {
	buff := &bytes.Buffer{}
	err := binary.Write(buff, binary.BigEndian, src)
	if err != nil {
		panic(err)
	}
	return buff.Bytes()
}

func addEntryToChan(avChan chan<- *container, b *utils.Buffer) {
	// get container address and port and add it to the channel
	bSlice := b.Buff()
	tMsg := getMsgFromBytes(bSlice[:])

	ipAddr := net.IPv4(tMsg.Addr[0], tMsg.Addr[1], tMsg.Addr[2], tMsg.Addr[3])
	cnt := &container{&ipAddr, tMsg.Port}
	log.Printf("Accepting incoming PING request from address %s port %d\n", ipAddr, tMsg.Port)
	avChan <- cnt

	// release the acquired buffer as soon as it is not useful anymore
	b.Release()
}

func AcceptRegisterRequests(avChan chan<- *container, addr *net.UDPAddr) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		log.Println("Error opening UDP connection on interface %s and port %d", addr.IP, addr.Port)
		return
	}
	defer conn.Close()

	bp := utils.NewBuffersPool(udp.MAX_UDP_PACKET_SIZE, 100)
	for {
		b := bp.Next()
		conn.Read(b.Buff())
		go addEntryToChan(avChan, b)
	}
}

func sendToContainer(cm *ContainerMap, incPkt <-chan []byte, avChan <-chan *container, log *log.Logger) {
	for {
		select {
		case pkt := <-incPkt:
			code, err := udp.PktCrc16(pkt)
			if err != nil {
				log.Printf("Error during calculation of CRC 16")
				continue
			}
			cnt, ok := cm.Get(code)
			if !ok {
				cnt = <-avChan
				go cm.Add(code, cnt)
			}
			conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*cnt.addr, int(cnt.port), ""})
			if err != nil {
				log.Printf("Error sending message to %s:%d: %s\n", *cnt.addr, int(cnt.port), err)
			}
			_, err = conn.Write(pkt)
			if err != nil {
				log.Printf("Error writing packet to socket: %s\n", err)
				conn.Close()
				continue
			}
			err = conn.Close()
			if err != nil {
				log.Printf("Error closing socket: %s\n", err)
				continue
			}
			log.Printf("Sent packet to %s:%d\n", (*cnt.addr), cnt.port)
			continue
		}
	}
}

func instantiateFunctions(hostname, auth, actionName string, instances int) {
	timeout := time.NewTicker(ACTION_TRIGGER_TIMEOUT_IN_SECONDS * time.Second)

	for {
		for i := 0; i < instances; i++ {
			err := op.CreateFunction(hostname, auth, actionName)
			if err != nil {
				log.Printf("Error creating function on OpenWhisk at hostname %s for action %s", hostname,
					actionName)
				log.Printf("Error obtained: %s", err)
			} else {
				log.Printf("Function %d created on OpenWhisk at %s with action %s", i+1, hostname, actionName)
			}
		}
		<-timeout.C
	}
}

func Handler(incPkt <-chan []byte, stopChan <-chan struct{}, hostname, auth, actionName string, logger *log.Logger) {
	cntChan := make(chan *container)
	cm := NewContainerMap(time.Duration(60 * time.Second))

	go instantiateFunctions(hostname, auth, actionName, 5)
	go AcceptRegisterRequests(cntChan, &net.UDPAddr{net.IPv4(0, 0, 0, 0), 9082, ""})

	for i := 0; i < MAX_SENDER; i++ {
		go sendToContainer(cm, incPkt, cntChan, logger)
	}

	<-stopChan
}
