package cnt

import (
	"bytes"
	"encoding/binary"
	op "faasrouter/openwhisk"
	"faasrouter/udp"
	"faasrouter/utils"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"time"

	"bitbucket.org/Manaphy91/nflib"
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

func addEntryToChan(cl *ContainerList, b *utils.Buffer) {
	// get container address and port and add it to the channel
	bSlice := b.Buff()
	tMsg := nflib.GetMsgFromBytes(bSlice[:])

	ipAddr := net.IPv4(tMsg.Addr[0], tMsg.Addr[1], tMsg.Addr[2], tMsg.Addr[3])
	cnt := &container{&ipAddr, tMsg.Port}
	utils.RLogger.Printf("Accepting incoming PING request from address %s port %d\n", ipAddr, tMsg.Port)
	cl.AddContainer(cnt)

	// release the acquired buffer as soon as it is not useful anymore
	b.Release()
}

func AcceptRegisterRequests(cl *ContainerList, addr *net.UDPAddr) {
	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		utils.RLogger.Println("Error opening UDP connection on interface %s and port %d", addr.IP, addr.Port)
		return
	}
	defer conn.Close()

	bp := utils.NewBuffersPool(udp.MAX_UDP_PACKET_SIZE, 100)
	for {
		b := bp.Next()
		conn.Read(b.Buff())
		go addEntryToChan(cl, b)
	}
}

func sendToContainer(cm *ContainerMap, incPkt <-chan []byte, cntChan <-chan *container, log *log.Logger) {
	for {
		select {
		case pkt := <-incPkt:
			code, err := udp.PktCrc16(pkt)
			if err != nil {
				utils.RLogger.Printf("Error during calculation of CRC 16")
				continue
			}
			cnt, ok := cm.Get(code)
			if !ok {
				cnt = <-cntChan
				go cm.Add(code, cnt)
			}
			conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*cnt.addr, int(cnt.port), ""})
			if err != nil {
				utils.RLogger.Printf("Error sending message to %s:%d: %s\n", *cnt.addr, int(cnt.port), err)
			}
			_, err = conn.Write(pkt)
			if err != nil {
				utils.RLogger.Printf("Error writing packet to socket: %s\n", err)
				conn.Close()
				continue
			}
			err = conn.Close()
			if err != nil {
				utils.RLogger.Printf("Error closing socket: %s\n", err)
				continue
			}
			utils.RLogger.Printf("Sent packet to %s:%d\n", (*cnt.addr), cnt.port)
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
				utils.RLogger.Printf("Error creating function on OpenWhisk at hostname %s for action %s", hostname,
					actionName)
				utils.RLogger.Printf("Error obtained: %s", err)
			} else {
				utils.RLogger.Printf("Function %d created on OpenWhisk at %s with action %s", i+1, hostname, actionName)
			}
		}
		<-timeout.C
	}
}

func handleContainerPool(cl *ContainerList, out chan<- *container) {
	for {
		for _, val := range cl.lst {
			out <- val
		}
	}
}

func ActionHandler(incPkt <-chan []byte, stopChan <-chan struct{}, hostname, auth, actionName string, logger *log.Logger) {
	cntChan := make(chan *container)
	cm := NewContainerMap(time.Duration(60 * time.Second))
	cl := NewContainerList()

	go handleContainerPool(cl, cntChan)

	go instantiateFunctions(hostname, auth, actionName, 5)
	go AcceptRegisterRequests(cl, &net.UDPAddr{net.IPv4(0, 0, 0, 0), 9082, ""})
	go handleContainerPool(cl, cntChan)

	for i := 0; i < MAX_SENDER; i++ {
		go sendToContainer(cm, incPkt, cntChan, logger)
	}

	<-stopChan
}

func InitRuleMap(stopChan <-chan struct{}, hostname, auth string, logger *log.Logger) *utils.RuleMap {
	rl := utils.NewRuleMap()

	dhcpChan := make(chan []byte, 200)

	rl.Add(func(pkt []byte) bool {
		_, trg, err := utils.GetPortsFromPkt(pkt)
		if err != nil {
			utils.RLogger.Printf("Error reading ports from the incoming packet: %s\n")
			return false
		}

		// check if the incoming message is a DNS message
		return trg == 68
	}, dhcpChan)

	go ActionHandler(dhcpChan, stopChan, hostname, auth, "dhcp", logger)

	natChan := make(chan []byte, 200)

	rl.Add(func(pkt []byte) bool {
		src, _, err := utils.GetPortsFromPkt(pkt)
		if err != nil {
			utils.RLogger.Printf("Error reading ports from the incoming packet: %s\n")
			return false
		}

		// filter out incoming messages having 53 as source port
		return src != 53
	}, natChan)

	go ActionHandler(natChan, stopChan, hostname, auth, "nat", logger)

	return rl
}
