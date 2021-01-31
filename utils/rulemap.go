package utils

import (
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/howeyc/crc16"
)

type Container struct {
	Addr   *net.IP
	Port   uint16
	Fluxes int8
	Mu     sync.RWMutex
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

type Crc2ConnMap struct {
	mu              sync.RWMutex
	crc2conn        map[uint16]*net.UDPConn
	connPool        *ConnPool
	crcFluxUsageMap map[uint16]int64
}

func NewCrc2ConnMap() (*Crc2ConnMap, *ConnPool) {
	res := new(Crc2ConnMap)
	res.crc2conn = make(map[uint16]*net.UDPConn)
	res.connPool = NewConnPool(res)
	res.crcFluxUsageMap = make(map[uint16]int64)
	return res, res.connPool
}

func (c2cm *Crc2ConnMap) Get(crc uint16) *net.UDPConn {
	c2cm.mu.RLock()
	val, ok := c2cm.crc2conn[crc]
	c2cm.mu.RUnlock()
	c2cm.mu.Lock()
	c2cm.crcFluxUsageMap[crc] = time.Now().UnixNano()
	c2cm.mu.Unlock()
	if ok {
		return val
	} else {
		cpe := c2cm.connPool.Next()
		cpe.AddCrc(crc)
		c2cm.mu.Lock()
		c2cm.crc2conn[crc] = cpe.conn
		c2cm.mu.Unlock()
		cpe.container.IncFluxes()
		go func(timeout time.Duration) {
			t := time.NewTicker(timeout)
			for {
				<-t.C
				c2cm.mu.RLock()
				if timestamp, okT := c2cm.crcFluxUsageMap[crc]; okT {
					c2cm.mu.RUnlock()
					deltaT := time.Now().UnixNano() - timestamp
					if deltaT > int64(timeout) {
						cpe.container.DecFluxes()
						c2cm.mu.Lock()
						delete(c2cm.crc2conn, crc)
						c2cm.mu.Unlock()
						return
					}
				}
			}
		}(1 * time.Millisecond)
		return cpe.conn
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
}

func NewConnPool(crc2ConnMap *Crc2ConnMap) *ConnPool {
	res := new(ConnPool)
	res.crc2ConnMap = crc2ConnMap
	return res
}

func (cp *ConnPool) Add(conn *net.UDPConn, container *Container) *ConnPoolElem {
	elem := &ConnPoolElem{conn, nil, nil, &cp.mu, cp, cp.crc2ConnMap, make(map[uint16]bool), container}
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
	if cp.headElem == nil {
		return nil
	} else {
		cp.mu.Lock()
		res := cp.currElem
		cp.currElem = res.next
		cp.mu.Unlock()
		return res
	}
}

func (ce *ConnPoolElem) AddCrc(crc uint16) {
	ce.mu.Lock()
	ce.ports[crc] = true
	ce.mu.Unlock()
}

func (ce *ConnPoolElem) Delete() {
	// current node no more available for new fluxes
	ce.mu.Lock()
	if ce.next == ce {
		ce.connPool.headElem = nil
		ce.connPool.tailElem = nil
		ce.connPool.currElem = nil
	} else {
		ce.prev.next, ce.next.prev = ce.next, ce.prev

		if ce.connPool.currElem == ce {
			ce.connPool.currElem = ce.next
		}

		if ce.connPool.headElem == ce {
			ce.connPool.headElem = ce.next
		}

		if ce.connPool.tailElem == ce {
			ce.connPool.tailElem = ce.next
		}
	}

	// used below as guard to avoid decrement after that the container has been deleted
	ce.next = nil
	ce.mu.Unlock()

	// safetly close connection after function return
	// defer ce.conn.Close()

	if ce.ports == nil || len(ce.ports) == 0 {
		// in case no connections are in place, close the opened one
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
		ce.container.IncFluxes()

		// schedule number of fluxes revision for replacement container
		go func(timeout time.Duration) {
			t := time.NewTicker(timeout)
			// loop til the container is available
			for ce.next != nil {
				<-t.C
				ce.crc2ConnMap.mu.RLock()
				if timestamp, okT := ce.crc2ConnMap.crcFluxUsageMap[port]; okT {
					ce.crc2ConnMap.mu.RUnlock()
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
		}(1 * time.Millisecond)

	}
	ce.crc2ConnMap.mu.Unlock()

	rCe.mu.Lock()
	for key, _ := range ce.ports {
		rCe.ports[key] = true
	}
	rCe.mu.Unlock()
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

func (rm RuleMap) GetChan(pkt []byte) *net.UDPConn {
	for rule, crc2Conn := range rm {
		if (func([]byte) bool)(*rule)(pkt) {
			code, _ := PktCrc16(pkt)
			return crc2Conn.Get(code)
		}
	}

	return nil
}
