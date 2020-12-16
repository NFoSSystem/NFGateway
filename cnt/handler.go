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

//func addEntryToChan(outChan chan<- *container, cl *ContainerList, buff []byte, size int) {
func addEntryToChan(cil map[string]*ContainerInfo, buff []byte, size int) {
	// get container address and port and add it to the channel
	//bSlice := b.Buff()
	tMsg := nflib.GetMsgFromBytes(buff[:size])

	ipAddr := net.IPv4(tMsg.Addr[0], tMsg.Addr[1], tMsg.Addr[2], tMsg.Addr[3])
	cnt := &container{addr: &ipAddr, port: tMsg.Port, fluxes: 1}
	actionName := trimActionName(fmt.Sprintf("%s", tMsg.Name))

	utils.RLogger.Printf("Accepting incoming PING request from address %s port %d action %s\n", ipAddr, tMsg.Port, actionName)

	cil[actionName].cntChan <- cnt
	cil[actionName].cntLst.AddContainer(cnt)

	// release the acquired buffer as soon as it is not useful anymore
	//b.Release()
}

func trimActionName(actionName string) string {
	for i, c := range actionName {
		if c == rune(0) {
			return actionName[:i]
		}
	}
	return actionName
}

func AcceptRegisterRequests(cil map[string]*ContainerInfo, addr *net.TCPAddr) {
	lstn, err := net.ListenTCP("tcp", addr)
	if err != nil {
		utils.RLogger.Printf("Error opening TCP connection on interface %s and port %d: %s\n", addr.IP, addr.Port, err)
		return
	}
	defer lstn.Close()

	utils.RLogger.Println("DEBUG Opened TCP socket at %s:%d\n", addr.IP, addr.Port, err)
	for {
		conn, err := lstn.AcceptTCP()
		if err != nil {
			utils.RLogger.Printf("Error accepting TCP connection from %s:%d: %s\n", addr.IP, addr.Port, err)
			continue
		}
		buff := make([]byte, 65535)
		size, _ := conn.Read(buff)
		go addEntryToChan(cil, buff, size)
	}
}

func sendToContainer(cm *ContainerMap, incPkt <-chan []byte, cntChan <-chan *container, log *log.Logger) {
	for {
		select {
		case pkt := <-incPkt:
			utils.RLogger.Println("DEBUG 1 sendToContainer")
			code, err := udp.PktCrc16(pkt)
			if err != nil {
				utils.RLogger.Printf("Error during calculation of CRC 16")
				continue
			}
			utils.RLogger.Println("DEBUG 2 sendToContainer")
			cnt, ok := cm.Get(code)
			if !ok {
				utils.RLogger.Println("DEBUG 2.5 sendToContainer")
				cnt = <-cntChan
				utils.RLogger.Println("DEBUG 2.6 sendToContainer")
				go cm.Add(code, cnt)
			}
			utils.RLogger.Println("DEBUG 3 sendToContainer")
			conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*cnt.addr, int(cnt.port), ""})
			if err != nil {
				utils.RLogger.Printf("Error sending message to %s:%d: %s\n", *cnt.addr, int(cnt.port), err)
			}
			utils.RLogger.Println("DEBUG 4 sendToContainer")
			_, err = conn.Write(pkt)
			if err != nil {
				utils.RLogger.Printf("Error writing packet to socket: %s\n", err)
				conn.Close()
				continue
			}
			utils.RLogger.Println("DEBUG 5 sendToContainer")
			err = conn.Close()
			if err != nil {
				utils.RLogger.Printf("Error closing socket: %s\n", err)
				continue
			}
			utils.RLogger.Println("DEBUG 6 sendToContainer")
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

// func handleContainerPool(cl *ContainerList, out chan<- *container) {
// 	for {
// 		if cl.Empty() {
// 			<-time.NewTimer(time.Millisecond).C
// 		}

// 		for _, val := range cl.lst {
// 			out <- val
// 		}
// 	}
// }

func handleContainerPool(inChan <-chan *container, outChan chan<- *container) {
	for {
		select {
		case cnt := <-inChan:
			outChan <- cnt
		}
	}
}

func ActionHandler(incPkt <-chan []byte, stopChan <-chan struct{}, inChan <-chan *container, cl *ContainerList, hostname, auth, actionName string, logger *log.Logger) {
	cntChan := make(chan *container, 50)
	cm := NewContainerMap(time.Duration(60 * time.Second))

	go handleContainerPool(inChan, cntChan)

	p := new(Provisioner)
	p.Hostname = hostname
	p.Auth = auth
	p.Action = actionName
	p.Check = 100 * time.Millisecond
	p.Timeout = 55 * time.Second
	p.UpThr = 0.6
	p.DownThr = 0.3
	p.UpInc = 0.5
	p.DownInc = 0.4
	p.Min = 1
	p.Cl = cl

	go p.InstantiateFunctions()

	//go instantiateFunctions(hostname, auth, actionName, 1)

	for i := 0; i < MAX_SENDER; i++ {
		go sendToContainer(cm, incPkt, cntChan, logger)
	}

	<-stopChan
}

func InitRuleMap(stopChan <-chan struct{}, hostname, auth string, logger *log.Logger) *utils.RuleMap {
	rl := utils.NewRuleMap()
	natCl := NewContainerList()
	//dhcpCl := NewContainerList()

	natOutChan := make(chan *container, 50)
	//dhcpOutChan := make(chan *container, 50)

	cil := make(map[string]*ContainerInfo)
	cil["nat"] = &ContainerInfo{natOutChan, natCl}
	//cil["dhcp"] = &ContainerInfo{dhcpOutChan, dhcpCl}

	go AcceptRegisterRequests(cil, &net.TCPAddr{net.IPv4(0, 0, 0, 0), 9082, ""})

	// dhcpChan := make(chan []byte, 200)

	// rl.Add(func(pkt []byte) bool {
	// 	_, trg, err := utils.GetPortsFromPkt(pkt)
	// 	if err != nil {
	// 		utils.RLogger.Printf("Error reading ports from the incoming packet: %s\n")
	// 		return false
	// 	}

	// 	// check if the incoming message is a DNS message
	// 	return trg == 67
	// }, dhcpChan)

	// go ActionHandler(dhcpChan, stopChan, dhcpOutChan, dhcpCl, hostname, auth, "dhcp", logger)

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

	go ActionHandler(natChan, stopChan, natOutChan, natCl, hostname, auth, "nat", logger)

	return rl
}
