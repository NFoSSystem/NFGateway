package cnt

import (
	"bytes"
	"encoding/binary"
	"faasrouter/utils"
	"fmt"
	"log"
	"net"
	"strconv"
	"strings"
	"sync"
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
func addEntryToChan(cil map[string]*ContainerInfo, a2c *Action2Cep, buff []byte, size int) {
	// get container address and port and add it to the channel
	//bSlice := b.Buff()
	tMsg := nflib.GetMsgFromBytes(buff[:size])

	ipAddr := net.IPv4(tMsg.Addr[0], tMsg.Addr[1], tMsg.Addr[2], tMsg.Addr[3])
	actionName := trimActionName(fmt.Sprintf("%s", tMsg.Name))
	cnt := &utils.Container{Addr: &ipAddr, Port: tMsg.Port, Fluxes: 0, Id: tMsg.CntId, Repl: tMsg.Repl, MaxFluxes: cil[actionName].maxFluxes}
	connPoolElem := cil[actionName].cntPool.Add(UDPConnFactory(&ipAddr, int(tMsg.Port)), cnt)
	cil[actionName].cntLst.AddContainer(cnt)

	utils.RLogger.Printf("Accepting incoming PING request from address %s port %d action %s\n", ipAddr, tMsg.Port, actionName)

	cep := a2c.Get(actionName)
	cep.AddConnElem(tMsg.CntId, connPoolElem, cil[actionName].cntLst)

	// go func() {
	// 	time.AfterFunc((MAX_ACTION_RUNTIME_IN_SECONDS-5)*time.Second, func() {
	// 		utils.RLogger.Printf("Before deleting function %s at address %s\n", actionName, ipAddr.String())
	// 		cil[actionName].cntLst.RemoveContainer(cnt)
	// 		connPoolElem.Delete()
	// 		connPoolElem = nil
	// 		log.Printf("DEBUG After delete in async goroutine")
	// 	})
	// }()

	go func(stopChan chan struct{}) {
		timer := time.NewTimer((MAX_ACTION_RUNTIME_IN_SECONDS - 5) * time.Second)
		cntId := connPoolElem.GetContainer().Id

		select {
		case <-timer.C:
			utils.RLogger.Printf("Before deleting function %s at address %s\n", actionName, ipAddr.String())
			cil[actionName].cntLst.RemoveContainer(cnt)
			connPoolElem.Delete()
			connPoolElem = nil
		case <-stopChan:
			log.Printf("Received message to terminate async delete function for container %d\n", cntId)
			return
		}

	}(connPoolElem.GetStopChan())
	// release the acquired buffer as soon as it is not useful anymore
	//b.Release()
}

func UDPConnFactory(addr *net.IP, port int) *net.UDPConn {
	conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*addr, port, ""})
	if err != nil {
		utils.RLogger.Printf("Error opening UDP connection to %s:%d: %s\n", err)
		return nil
	}
	return conn
}

func trimActionName(actionName string) string {
	for i, c := range actionName {
		if c == rune(0) {
			return actionName[:i]
		}
	}
	return actionName
}

// func AcceptRegisterRequests(cil map[string]*ContainerInfo, cep *ConnElemProvisioner, addr *net.TCPAddr) {
// 	lstn, err := net.ListenTCP("tcp", addr)
// 	if err != nil {
// 		utils.RLogger.Printf("Error opening TCP connection on interface %s and port %d: %s\n", addr.IP, addr.Port, err)
// 		return
// 	}
// 	defer lstn.Close()

// 	for {
// 		conn, err := lstn.AcceptTCP()
// 		if err != nil {
// 			utils.RLogger.Printf("Error accepting TCP connection from %s:%d: %s\n", addr.IP, addr.Port, err)
// 			continue
// 		}
// 		buff := make([]byte, 65535)
// 		size, _ := conn.Read(buff)
// 		go addEntryToChan(cil, cep, buff, size)
// 	}
// }

func AcceptRegisterRequests(cil map[string]*ContainerInfo, a2c *Action2Cep, addr *net.TCPAddr) {
	lstn, err := net.ListenTCP("tcp", addr)
	if err != nil {
		utils.RLogger.Printf("Error opening TCP connection on interface %s and port %d: %s\n", addr.IP, addr.Port, err)
		return
	}
	defer lstn.Close()

	for {
		conn, err := lstn.AcceptTCP()
		if err != nil {
			utils.RLogger.Printf("Error accepting TCP connection from %s:%d: %s\n", addr.IP, addr.Port, err)
			continue
		}
		buff := make([]byte, 65535)
		size, _ := conn.Read(buff)
		go addEntryToChan(cil, a2c, buff, size)
	}
}

// func sendToContainer(cm *ContainerMap, incPkt <-chan []byte, cntChan <-chan *container, log *log.Logger) {
// 	for {
// 		select {
// 		case pkt := <-incPkt:
// 			code, err := udp.PktCrc16(pkt)
// 			if err != nil {
// 				utils.RLogger.Printf("Error during calculation of CRC 16")
// 				continue
// 			}
// 			cnt, ok := cm.Get(code)
// 			if !ok {
// 				cnt = <-cntChan
// 				go cm.Add(code, cnt)
// 			}
// 			conn, err := net.DialUDP("udp", nil, &net.UDPAddr{*cnt.addr, int(cnt.port), ""})
// 			if err != nil {
// 				utils.RLogger.Printf("Error sending message to %s:%d: %s\n", *cnt.addr, int(cnt.port), err)
// 			}
// 			_, err = conn.Write(pkt)
// 			if err != nil {
// 				utils.RLogger.Printf("Error writing packet to socket: %s\n", err)
// 				conn.Close()
// 				continue
// 			}
// 			err = conn.Close()
// 			if err != nil {
// 				utils.RLogger.Printf("Error closing socket: %s\n", err)
// 				continue
// 			}
// 			continue
// 		}
// 	}
// }

func sendToContainerHelper(pkt []byte, crcConnMap map[uint16]*net.UDPConn, mutex sync.RWMutex, cntChan <-chan *utils.Container) {
	code, err := utils.PktCrc16(pkt)
	if err != nil {
		utils.RLogger.Printf("Error during calculation of CRC 16")
		return
	}

	mutex.RLock()
	conn, ok := crcConnMap[code]
	mutex.RUnlock()
	if ok {
		_, err := conn.Write(pkt)
		if err != nil {
			utils.RLogger.Printf("Error writing packet to socket: %s\n", err)
			return
		}
	} else {
		mutex.Lock()
		// perform one more check
		conn, ok := crcConnMap[code]
		if !ok {
			cnt := <-cntChan
			conn, err = net.DialUDP("udp", nil, &net.UDPAddr{*cnt.Addr, int(cnt.Port), ""})
			if err != nil {
				utils.RLogger.Printf("Error sending message to %s:%d: %s\n", *cnt.Addr, int(cnt.Port), err)
			}
			crcConnMap[code] = conn
		}
		mutex.Unlock()
		_, err = conn.Write(pkt)
		if err != nil {
			utils.RLogger.Printf("Error writing packet to socket: %s\n", err)
			return
		}
	}
}

func SendToContainer3(pkt []byte, conn *net.UDPConn) {
	_, err := conn.Write(pkt)
	if err != nil {
		utils.RLogger.Printf("Error sending message to UDP socket: %s - conn: %v\n", err, conn)
	}
}

func sendToContainer(cm *ContainerMap, incPkt <-chan []byte, cntChan <-chan *utils.Container, log *log.Logger, mutex sync.RWMutex, crcConnMap map[uint16]*net.UDPConn) {
	for {
		select {
		case pkt := <-incPkt:
			go sendToContainerHelper(pkt, crcConnMap, mutex, cntChan)
		}
	}
}

// func instantiateFunctions(hostname, auth, actionName, redisIp string, redisPort int, instances int) {
// 	timeout := time.NewTicker(ACTION_TRIGGER_TIMEOUT_IN_SECONDS * time.Second)

// 	for {
// 		for i := 0; i < instances; i++ {
// 			err := op.CreateFunction(hostname, auth, actionName, redisIp, redisPort)
// 			if err != nil {
// 				utils.RLogger.Printf("Error creating function on OpenWhisk at hostname %s for action %s", hostname,
// 					actionName)
// 				utils.RLogger.Printf("Error obtained: %s", err)
// 			} else {
// 				utils.RLogger.Printf("Function %d created on OpenWhisk at %s with action %s", i+1, hostname, actionName)
// 			}
// 		}
// 		<-timeout.C
// 	}
// }

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

func handleContainerPool(inChan <-chan *utils.Container, outChan chan<- *utils.Container) {
	for {
		select {
		case cnt := <-inChan:
			outChan <- cnt
		}
	}
}

func ActionHandler(stopChan <-chan struct{}, cl *ContainerList, hostname, auth, actionName, redisIp string, redisPort int, logger *log.Logger, cep *ConnElemProvisioner) {
	p := new(Provisioner)
	p.Hostname = hostname
	p.Auth = auth
	p.Action = actionName
	p.Check = 3 * time.Second
	p.Timeout = 45 * time.Second
	p.UpThr = 0.6
	p.DownThr = 0.3
	p.UpInc = 0.5
	p.DownInc = 0.4
	p.Min = 1
	p.Cl = cl
	p.RedisIp = redisIp
	p.RedisPort = redisPort
	p.MaxFluxes = 2

	go p.InstantiateFunctions(cep)

	//go instantiateFunctions(hostname, auth, actionName, 1)

	<-stopChan
}

func InitRuleMap(stopChan <-chan struct{}, hostname, auth string, logger *log.Logger, redisIp string, redisPort int) *utils.RuleMap {
	rl := utils.NewRuleMap()
	natCl := NewContainerList()
	//dhcpCl := NewContainerList()

	natCrc2ConnMap, natConnPool := utils.NewCrc2ConnMap(time.Duration(500 * time.Microsecond))
	// dhcpCrc2ConnMap, dhcpConnPool := utils.NewCrc2ConnMap(time.Duration(100 * time.Millisecond))

	cil := make(map[string]*ContainerInfo)

	cil["nat"] = &ContainerInfo{natConnPool, natCl, 10}
	natCep := NewConnElemProvisioner()

	// cil["dhcp"] = &ContainerInfo{dhcpConnPool, dhcpCl, 2}
	// dhcpCep := NewConnElemProvisioner()

	a2c := NewAction2Cep()
	a2c.Set("nat", natCep)
	//a2c.Set("dhcp", dhcpCep)

	go AcceptRegisterRequests(cil, a2c, &net.TCPAddr{net.IPv4(0, 0, 0, 0), 9082, ""})

	// rl.Add(func(pkt []byte) bool {
	// 	_, trg, err := utils.GetPortsFromPkt(pkt)
	// 	if err != nil {
	// 		utils.RLogger.Printf("Error reading ports from the incoming packet: %s\n")
	// 		return false
	// 	}

	// 	// check if the incoming message is a DHCP message
	// 	return trg == 67
	// }, dhcpCrc2ConnMap)

	// go ActionHandler(stopChan, dhcpCl, hostname, auth, "dhcp", redisIp, redisPort, logger, dhcpCep)

	rl.Add(func(pkt []byte) bool {
		src, _, err := utils.GetPortsFromPkt(pkt)
		if err != nil {
			utils.RLogger.Printf("Error reading ports from the incoming packet: %s\n")
			return false
		}

		// filter out incoming messages having 53 as source port
		return src != 53
	}, natCrc2ConnMap)

	go ActionHandler(stopChan, natCl, hostname, auth, "nat", redisIp, redisPort, logger, natCep)

	return rl
}
