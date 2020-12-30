package nat

import (
	"io"
	"log"
	"net"
	"sync"
	"time"

	"bitbucket.org/Manaphy91/nflib"
	"github.com/willf/bitset"
)

const (
	TCP_MAPPING_REQUEST_PORT = 8923
)

type FuncCntMapping map[uint32]*uint32

type NatCouple struct {
	Cc16 uint16
	Port uint16
}

type CntPortMapping struct {
	mu   sync.RWMutex
	bSet *bitset.BitSet // contains the list of used ports
	im   map[uint32][]*NatCouple
	am   map[*uint32][]uint16 // assignment map, it contains the assignment buffer obtained using the bSet BitSet
}

func setHelper(offset uint16, acc []uint16, cnt *bitset.BitSet, stop chan []uint16) {
	idx, avail := cnt.NextClear(uint(offset))
	if !avail {
		stop <- acc
	}
	length := idx + 100
	i := 0
	for ; idx < length; idx++ {
		if !cnt.Test(idx) {
			cnt.Set(idx)
			acc[i] = uint16(idx)
			i++
		}
	}
	stop <- acc
}

func NewCntPortMapping() *CntPortMapping {
	res := new(CntPortMapping)
	res.bSet = bitset.New(65536)
	res.im = make(map[uint32][]*NatCouple)
	res.am = make(map[*uint32][]uint16)
	return res
}

func (cp *CntPortMapping) AssignPorts(cntAddr *uint32) []uint16 {
	cp.mu.Lock()
	lower, upper := make([]uint16, 100), make([]uint16, 100)
	sLChan, sUChan := make(chan []uint16), make(chan []uint16)
	go setHelper(1, lower, cp.bSet, sLChan)
	go setHelper(1025, upper, cp.bSet, sUChan)
	lower = <-sLChan
	upper = <-sUChan
	res := append(lower, upper...)
	cp.am[cntAddr] = res
	cp.mu.Unlock()
	return res
}

func releaseHelper(port []uint16, checkMap map[uint16]*uint32, counter *bitset.BitSet) {
	for i := 0; i < len(port); i++ {
		if _, ok := checkMap[uint16(port[i])]; !ok {
			counter.Clear(uint(port[i]))
		}
	}
}

func (cp *CntPortMapping) ReleasePorts(cntAddr *uint32, pc *PortCntMapping) {
	cp.mu.Lock()
	buff := cp.am[cntAddr]
	if buff == nil {
		cp.mu.Unlock()
		return
	}
	pc.mu.RLock()
	releaseHelper(buff, pc.im, cp.bSet)
	pc.mu.RUnlock()
	cp.mu.Unlock()
}

func (cp *CntPortMapping) Get(cntAddr uint32) []*NatCouple {
	cp.mu.RLock()
	if val, ok := cp.im[cntAddr]; !ok {
		cp.mu.RUnlock()
		return nil
	} else {
		cp.mu.RUnlock()
		return val
	}
}

func (cp *CntPortMapping) Set(cntAddr uint32, nc *NatCouple) {
	cp.mu.Lock()
	if val, ok := cp.im[cntAddr]; !ok {
		cp.im[cntAddr] = append(val, nc)
		cp.mu.Unlock()
	} else {
		lst := make([]*NatCouple, 1)
		lst = append(lst, nc)
		cp.im[cntAddr] = lst
		cp.mu.Unlock()
	}
}

func (cp *CntPortMapping) Delete(cntAddr uint32) {
	cp.mu.Lock()
	delete(cp.im, cntAddr)
	cp.mu.Unlock()
}

type PortCntMapping struct {
	mu           sync.RWMutex
	im           map[uint16]*uint32
	rm           map[uint32]*uint32   // reverse map for fast port re-allocation in O(1)
	lum          map[uint16]time.Time // mapping between port and last usage timestamp
	leaseTimeout time.Duration        // max lease time for a port
}

func NewPortCntMapping(timeout time.Duration) *PortCntMapping {
	res := new(PortCntMapping)
	res.im = make(map[uint16]*uint32)
	res.rm = make(map[uint32]*uint32)
	res.lum = make(map[uint16]time.Time)
	res.leaseTimeout = timeout
	return res
}

func (pc *PortCntMapping) Get(port uint16) uint32 {
	pc.mu.RLock()
	if val, ok := pc.im[port]; !ok {
		pc.mu.RUnlock()
		return 0
	} else {
		pc.mu.RUnlock()
		return *val
	}
}

func (pc *PortCntMapping) Set(port uint16, addr uint32) {
	pc.mu.Lock()
	if val, ok := pc.rm[addr]; !ok {
		addrPtr := addr
		pc.rm[addr] = &addrPtr
		pc.im[port] = &addrPtr
	} else {
		pc.im[port] = val
	}
	pc.lum[port] = time.Now()
	pc.mu.Unlock()

	time.AfterFunc(pc.leaseTimeout, func() {
	funcStart:
		pc.mu.RLock()
		if pc.lum[port].Add(pc.leaseTimeout).Before(time.Now()) {
			pc.mu.RUnlock()
			// the port lease expired, remove it from the maps
			pc.mu.Lock()
			delete(pc.im, port)
			delete(pc.lum, port)
			pc.mu.Unlock()
		} else {
			pc.mu.RUnlock()
			time.Sleep(pc.leaseTimeout)
			goto funcStart
		}
	})
}

func (pc *PortCntMapping) Replace(oldAddr, newAddr uint32) {
	pc.mu.Lock()
	if val, ok := pc.rm[oldAddr]; ok {
		*val = newAddr
	}
	pc.mu.Unlock()
}

func (pc *PortCntMapping) Delete(port uint16) {
	pc.mu.Lock()
	delete(pc.im, port)
	pc.mu.Unlock()
}

func ListenForMappingRequests(cp *CntPortMapping, pc *PortCntMapping) {
	lstn, err := net.ListenTCP("tcp", &net.TCPAddr{net.IPv4(0, 0, 0, 0), TCP_MAPPING_REQUEST_PORT, ""})
	if err != nil {
		log.Fatalf("Error opening TCP socket on port %d: %s\n", TCP_MAPPING_REQUEST_PORT, err)
	}
	defer lstn.Close()

	buff := make([]byte, 1500)

	for {
		conn, err := lstn.AcceptTCP()
		if err != nil {
			log.Fatalf("Error accepting TCP connection: %s\n", err)
		}

		size, err := conn.Read(buff)
		if err == io.EOF {
			continue
		} else if err != nil {
			log.Fatalf("Error reading from TCP socket: %s\n", err)
		}

		pkt := nflib.GetPacketFromBytes(buff[:size])

		go func(pkt *nflib.Packet) {
			nc := &NatCouple{pkt.Crc16, pkt.Port}
			cp.Set(pkt.Addr, nc)
			pc.Set(pkt.Port, pkt.Addr)
		}(pkt)
	}
}
