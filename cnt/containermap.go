package cnt

import (
	"net"
	"sync"
	"time"
)

type container struct {
	addr *net.IP
	port uint16
}

type ContainerMap struct {
	im      map[uint16]*container
	mu      sync.RWMutex
	timeout *time.Duration
}

func NewContainerMap(timeout time.Duration) *ContainerMap {
	res := new(ContainerMap)
	res.im = make(map[uint16]*container)
	res.timeout = &timeout
	return res
}

func cleanEntry(cp *ContainerMap, key uint16, timerPtr <-chan time.Time, stop <-chan struct{}) {
	select {
	case <-stop:
		return
	case <-timerPtr:
		cp.mu.Lock()
		delete(cp.im, key)
		cp.mu.Unlock()
		return
	}
}

func (cpm *ContainerMap) Add(crc uint16, cnt *container) {
	cpm.mu.Lock()
	timer := time.NewTimer(*(cpm.timeout)).C
	stop := make(chan struct{})
	go cleanEntry(cpm, crc, timer, stop)
	cpm.im[crc] = cnt
	cpm.mu.Unlock()
}

func (cpm *ContainerMap) Get(crc uint16) (*container, bool) {
	cpm.mu.Lock()
	defer cpm.mu.Unlock()
	if res, ok := cpm.im[crc]; ok {
		return res, ok
	} else {
		return nil, false
	}
}
