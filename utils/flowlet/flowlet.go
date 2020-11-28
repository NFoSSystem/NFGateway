package flowlet

import (
	"sync"
	"time"
)

type Flowlet struct {
	size    uint16
	latency time.Duration
}

type FlowletMap struct {
	maxSize    uint16
	maxLatency time.Duration
	innMap     map[uint16]*Flowlet
	mux        sync.RWMutex
}

/*
NewFlowletMap takes sizeLimt and latencyLimit as parameters and provide a FlowletMap instance
*/
func NewFlowletMap(sizeLimit uint16, latencyLimit time.Duration) *FlowletMap {
	flMap := make(map[uint16]*Flowlet)

	return &FlowletMap{innMap: flMap}
}

func (fm *FlowletMap) AddInstance(routeInfo []byte, size uint16, timing time.Duration) {
	fm.mux.Lock()

	fm.mux.Unlock()
}
