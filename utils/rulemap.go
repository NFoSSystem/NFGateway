package utils

import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/howeyc/crc16"
	"golang.org/x/sync/semaphore"
)

const (
	MAX_ACTION_RUNTIME_IN_SECONDS = 60
)

type Container struct {
	Addr      *net.IP
	Port      uint16
	Fluxes    int8
	Mu        sync.RWMutex
	Id        uint16
	Repl      bool
	MaxFluxes int8
}

func (c *Container) IncFluxes() {
	c.Mu.Lock()
	c.Fluxes += 1
	c.Mu.Unlock()
}

func (c *Container) DecFluxes() {
	c.Mu.Lock()
	if c.Fluxes > 1 {
		c.Fluxes -= 1
	} else {
		c.Fluxes = 0
	}
	c.Mu.Unlock()
}

func (c *Container) GetFluxes() int8 {
	c.Mu.RLock()
	res := c.Fluxes
	c.Mu.RUnlock()
	return res
}

func (c *Container) GetMaxFluxes() int8 {
	return c.MaxFluxes
}

type Crc2ConnMap struct {
	mu              sync.RWMutex
	crc2conn        map[uint16]*net.UDPConn
	crc2lock        map[uint16]*semaphore.Weighted
	connPool        *ConnPool
	crcFluxUsageMap map[uint16]int64
	crcContainer    map[uint16]*Container
	fluxMaxAge      time.Duration
}

func NewCrc2ConnMap(fluxMaxAge time.Duration) (*Crc2ConnMap, *ConnPool) {
	res := new(Crc2ConnMap)
	res.crc2conn = make(map[uint16]*net.UDPConn)
	res.crc2lock = make(map[uint16]*semaphore.Weighted)
	res.connPool = NewConnPool(res)
	res.crcFluxUsageMap = make(map[uint16]int64)
	res.crcContainer = make(map[uint16]*Container)
	res.fluxMaxAge = fluxMaxAge
	return res, res.connPool
}

func (c2cm *Crc2ConnMap) Get(crc uint16) (*net.UDPConn, *semaphore.Weighted) {
	c2cm.mu.RLock()
	val, ok := c2cm.crc2conn[crc]
	container, okCnt := c2cm.crcContainer[crc]
	lock, _ := c2cm.crc2lock[crc]
	c2cm.mu.RUnlock()
	c2cm.mu.Lock()
	c2cm.crcFluxUsageMap[crc] = time.Now().UnixNano()
	c2cm.mu.Unlock()

	// c2cm.mu.Lock()
	// val, ok := c2cm.crc2conn[crc]
	// container, okCnt := c2cm.crcContainer[crc]
	// lock, _ := c2cm.crc2lock[crc]
	// c2cm.crcFluxUsageMap[crc] = time.Now().UnixNano()
	// c2cm.mu.Unlock()
	if ok && okCnt && container.GetFluxes() <= container.GetMaxFluxes() {
		//log.Printf("DEBUG c2cm: %v\n", c2cm)
		return val, lock
	} else {
		// log.Printf("DEBUG ok %v - okCnt %v - crc %v\n", ok, okCnt, crc)
		// if container != nil {
		// 	log.Printf("DEBUG fluxes %v - max fluxes %v\n", container.GetFluxes(), container.GetMaxFluxes())
		// }
		cpe := c2cm.connPool.Next()
		cpe.AddCrc(crc)
		c2cm.mu.Lock()
		// if val, ok = c2cm.crc2conn[crc]; ok {
		// 	c2cm.mu.Unlock()
		// 	return val, c2cm.crc2lock[crc]
		// }
		c2cm.crc2conn[crc] = cpe.conn
		sem := semaphore.NewWeighted(1000000)
		c2cm.crc2lock[crc] = sem
		c2cm.crcContainer[crc] = cpe.container
		c2cm.mu.Unlock()
		cpe.container.IncFluxes()
		go func(timeout time.Duration) {
			t := time.NewTicker(timeout)
			for {
				select {
				case <-cpe.stopChan:
					return
				case <-t.C:
					c2cm.mu.RLock()
					timestamp, okT := c2cm.crcFluxUsageMap[crc]
					c2cm.mu.RUnlock()
					if okT {
						deltaT := time.Now().UnixNano() - timestamp
						if deltaT > int64(timeout) {
							cpe.container.DecFluxes()
							c2cm.mu.Lock()
							delete(c2cm.crc2conn, crc)
							delete(c2cm.crcContainer, crc)
							c2cm.mu.Unlock()
							return
						}
					}
				}
			}
		}(c2cm.fluxMaxAge)
		//}(1 * time.Millisecond)
		return cpe.conn, sem
	}
}

type ConnPool struct {
	mu          sync.RWMutex
	currElem    *ConnPoolElem
	headElem    *ConnPoolElem
	tailElem    *ConnPoolElem
	crc2ConnMap *Crc2ConnMap
}

type ConnPoolElem struct {
	conn        *net.UDPConn
	prev        *ConnPoolElem
	next        *ConnPoolElem
	mu          *sync.RWMutex
	connPool    *ConnPool
	crc2ConnMap *Crc2ConnMap
	ports       map[uint16]bool
	container   *Container
	stopChan    chan struct{}
}

func NewConnPool(crc2ConnMap *Crc2ConnMap) *ConnPool {
	res := new(ConnPool)
	res.crc2ConnMap = crc2ConnMap
	return res
}

func (cp *ConnPool) Size() int {
	i := 0
	log.Println("Size rlock")
	cp.mu.RLock()
	currElem := cp.headElem
	for currElem != cp.tailElem {
		if i > 5 {
			break
		}
		currElem = currElem.next
		i++
	}
	cp.mu.RUnlock()
	log.Println("Size runlock")
	return i
}

func (cp *ConnPool) SizeNoLock() int {
	if cp.headElem == nil {
		return 0
	}

	i := 1
	currElem := cp.headElem
	for currElem != cp.tailElem {
		currElem = currElem.next
		i++
	}
	return i
}

func (cp *ConnPool) Add(conn *net.UDPConn, container *Container) *ConnPoolElem {
	elem := &ConnPoolElem{conn, nil, nil, &cp.mu, cp, cp.crc2ConnMap, make(map[uint16]bool), container, make(chan struct{})}
	//time.Now().UnixNano()}
	cp.mu.Lock()
	if cp.headElem == nil {
		// also tail is nil
		// then, the new element will be both head and tail
		cp.headElem = elem
		cp.tailElem = elem
		elem.next = elem
		elem.prev = elem
		cp.currElem = cp.headElem
	} else {
		// the new element will be added right after the tail
		// and right before the head
		cp.tailElem.next = elem
		elem.prev = cp.tailElem
		elem.next = cp.headElem
		cp.headElem.prev = elem
		cp.tailElem = elem
	}

	cp.mu.Unlock()
	return elem
}

func (cp *ConnPool) Next() *ConnPoolElem {
	cp.mu.Lock()

	// var res *ConnPoolElem
	// for {
	// 	res = cp.currElem
	// 	cp.currElem = res.next
	// 	if (cp.currElem.addTime-time.Now().UnixNano())/int64(time.Second) <= (MAX_ACTION_RUNTIME_IN_SECONDS - 10) {
	// 		break
	// 	}
	// }

	res := cp.currElem
	cp.currElem = res.next

	// var headId, tailId, currId, size int = -1, -1, -1, -1
	// if cp.headElem != nil {
	// 	headId = int(cp.headElem.container.Id)
	// }

	// if cp.tailElem != nil {
	// 	tailId = int(cp.tailElem.container.Id)
	// }

	// if cp.currElem != nil {
	// 	currId = int(cp.currElem.container.Id)
	// }

	// size = cp.SizeNoLock()
	cp.mu.Unlock()

	//log.Printf("DEBUG Next from CP - cp.head %d - cp.tail %d - cp.curr %d - size %d", headId, tailId, currId, size)
	return res
}

func (ce *ConnPoolElem) AddCrc(crc uint16) {
	ce.mu.Lock()
	ce.ports[crc] = true
	ce.mu.Unlock()
}

func (ce *ConnPoolElem) GetFluxes() int8 {
	ce.mu.RLock()
	res := ce.container.Fluxes
	ce.mu.RUnlock()
	return res
}

func (ce *ConnPoolElem) GetContainer() *Container {
	ce.mu.RLock()
	res := ce.container
	ce.mu.RUnlock()
	return res
}

func (ce *ConnPoolElem) Delete() {
	// current node no more available for new fluxes
	// ce.connPool.mu.Lock()
	// if ce.next == ce {

	// 	// ce.connPool.currElem = nil
	// 	log.Println("DEBUG 1 currElem.cntId %d - connPool size %d - head.cntId %d - tail.cntId %d", ce.connPool.SizeNoLock(),
	// 		ce.connPool.currElem.container.Id, ce.connPool.headElem.container.Id, ce.connPool.tailElem.container.Id)

	// 	ce.connPool.headElem = nil
	// 	ce.connPool.tailElem = nil
	// } else {
	// 	if ce.prev == nil || ce.next == nil {
	// 		ce.conn = nil
	// 		ce.container = nil
	// 		ce.crc2ConnMap = nil
	// 		ce.connPool.mu.Unlock()
	// 		return
	// 	} else {
	// 		ce.prev.next, ce.next.prev = ce.next, ce.prev

	// 		if ce.connPool.currElem == ce {
	// 			ce.connPool.currElem = ce.next
	// 		}

	// 		if ce.connPool.headElem == ce {
	// 			ce.connPool.headElem = ce.next
	// 		}

	// 		if ce.connPool.tailElem == ce {
	// 			ce.connPool.tailElem = ce.next
	// 		}
	// 	}
	// }

	// var headId, tailId, currId, size int = -1, -1, -1, -1
	// if ce.connPool.headElem != nil {
	// 	headId = int(ce.connPool.headElem.container.Id)
	// }

	// if ce.connPool.tailElem != nil {
	// 	tailId = int(ce.connPool.tailElem.container.Id)
	// }

	// if ce.connPool.currElem != nil {
	// 	currId = int(ce.connPool.currElem.container.Id)
	// }

	// size = ce.connPool.SizeNoLock()
	// ce.connPool.mu.Unlock()

	ce.connPool.mu.Lock()
	if ce.next == ce {

		// ce.connPool.currElem = nil
		ce.connPool.headElem = nil
		ce.connPool.tailElem = nil
	} else {
		if ce.prev == nil || ce.next == nil {
			ce.conn = nil
			ce.container = nil
			ce.crc2ConnMap = nil
			ce.connPool.mu.Unlock()
			return
		} else {
			ce.prev.next, ce.next.prev = ce.next, ce.prev

			if ce.connPool.currElem == ce {
				ce.connPool.currElem = ce.next
			}

			if ce.connPool.headElem == ce {
				ce.connPool.headElem = ce.next
			}

			if ce.connPool.tailElem == ce {
				ce.connPool.tailElem = ce.prev
			}
		}
	}
	ce.connPool.mu.Unlock()

	ce.mu.Lock()
	close(ce.stopChan)
	// used below as guard to avoid decrement after that the container has been deleted
	ce.next = nil
	ce.prev = nil
	//ce.container = nil

	ce.mu.Unlock()

	// safetly close connection after function return
	// defer ce.conn.Close()

	if ce.ports == nil || len(ce.ports) == 0 {
		// in case no connections are in place, close the opened one
		sendTerminationSignal(ce.conn, ce.container.Addr.String())
		ce.container = nil
		return
	}

	// use innerNext method to obtain the next available
	// ConnPoolElem with which you will replace the caller
	rCe := ce.connPool.Next()

	// reallocate other connections in Crc2ConnMap
	ce.crc2ConnMap.mu.Lock()

	for port, _ := range ce.ports {
		ce.crc2ConnMap.crc2conn[port] = rCe.conn

		// increment number of fluxes for replacement container
		rCe.container.IncFluxes()

		go func(timeout time.Duration, port uint16) {
			t := time.NewTicker(timeout)
			for {
				select {
				case <-rCe.stopChan:
					return
				case <-t.C:
					ce.crc2ConnMap.mu.RLock()
					timestamp, okT := ce.crc2ConnMap.crcFluxUsageMap[port]
					ce.crc2ConnMap.mu.RUnlock()
					if okT {
						deltaT := time.Now().UnixNano() - timestamp
						if deltaT > int64(timeout) {
							rCe.container.DecFluxes()
							ce.crc2ConnMap.mu.Lock()
							delete(ce.crc2ConnMap.crc2conn, port)
							ce.crc2ConnMap.mu.Unlock()
							return
						}
					}
				}
			}
		}(500*time.Microsecond, port)

	}
	ce.crc2ConnMap.mu.Unlock()

	rCe.mu.Lock()
	for key, _ := range ce.ports {
		rCe.ports[key] = true
	}
	rCe.mu.Unlock()

	sendTerminationSignal(ce.conn, ce.container.Addr.String())
}

func (ce *ConnPoolElem) IsAlive() bool {
	ce.mu.RLock()
	res := (ce.prev != nil)
	ce.mu.RUnlock()
	return res
}

func (ce *ConnPoolElem) IsRepl() bool {
	ce.mu.RLock()
	res := ce.container.Repl
	ce.mu.RUnlock()
	return res
}

func (ce *ConnPoolElem) GetStopChan() chan struct{} {
	ce.mu.RLock()
	res := ce.stopChan
	ce.mu.RUnlock()
	return res
}

func sendTerminationSignal(conn *net.UDPConn, ip string) {
	_, err := conn.Write([]byte("terminate"))
	if err != nil {
		RLogger.Printf("Error sending termination message to %s\n", ip)
	}

	RLogger.Printf("Termination message sent to %s\n", ip)
}

type ruleFunc *func([]byte) bool

type RuleMap map[ruleFunc]*Crc2ConnMap

func NewRuleMap() *RuleMap {
	rl := RuleMap(make(map[ruleFunc]*Crc2ConnMap))
	return &rl
}

func (rm RuleMap) Add(f func([]byte) bool, crc2Conn *Crc2ConnMap) {
	rf := ruleFunc(&f)
	rm[rf] = crc2Conn
}

/*
PktCrc16 calculates Crc16 from source IP, target IP, source port and target port from provided packet.
*/
func PktCrc16(buff []byte) (uint16, error) {
	bSlice := make([]byte, 12)

	srcIP, trgIP, err := GetIPsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading IP addresses from packet!")
	}

	src, trg, err := GetPortsFromPkt(buff)
	if err != nil {
		return uint16(0), fmt.Errorf("Error reading source and target addresses from packet!")
	}

	bSlice = append(bSlice, []byte(srcIP)...)
	bSlice = append(bSlice, []byte(trgIP)...)
	bSlice = append(bSlice, uint8(src>>8), uint8(src&255))
	bSlice = append(bSlice, uint8(trg>>8), uint8(trg&255))

	return crc16.ChecksumIBM(bSlice), nil
}

func (rm RuleMap) GetChan(pkt []byte) (*net.UDPConn, *semaphore.Weighted) {
	for rule, crc2Conn := range rm {
		if (func([]byte) bool)(*rule)(pkt) {
			code, _ := PktCrc16(pkt)
			return crc2Conn.Get(code)
		}
	}

	return nil, nil
}
