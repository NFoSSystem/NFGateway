package cnt

import (
	"nfgateway/utils"
	"sync"
)

type ContainerList struct {
	lst      []*utils.Container
	mu       sync.RWMutex
	duration uint64
}

func NewContainerList() *ContainerList {
	cl := new(ContainerList)
	cl.mu = sync.RWMutex{}
	return cl
}

func (cl *ContainerList) AddContainer(cnt *utils.Container) {
	cl.mu.Lock()
	cl.lst = append(cl.lst, cnt)
	cl.mu.Unlock()
}

func (cl *ContainerList) RemoveContainer(cnt *utils.Container) {
	cl.mu.Lock()
	for idx, val := range cl.lst {
		if val == cnt {
			if idx != len(cl.lst)-1 {
				cl.lst = append(cl.lst[:idx], cl.lst[idx+1:]...)
			} else {
				cl.lst = cl.lst[:idx]
			}
		}
	}
	cl.mu.Unlock()
}

func (cl *ContainerList) GetList() []*utils.Container {
	return cl.lst
}

func (cl *ContainerList) Empty() bool {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return len(cl.lst) == 0
}

func min(a, b int8) int8 {
	if a < b {
		return a
	} else {
		return b
	}
}

func (cl *ContainerList) GetContainer() *utils.Container {
	for {
		if cl.Empty() {
			continue
		}

		cl.mu.RLock()
		defer cl.mu.RUnlock()
		length := len(cl.lst)
		var i int
		for i = length - 1; i > 0; i-- {
			cl.lst[i].Mu.RLock()
			cl.lst[i-1].Mu.RLock()
			fMin := min(cl.lst[i].Fluxes, cl.lst[i-1].Fluxes)
			if fMin != cl.lst[i-1].Fluxes {
				defer cl.lst[i-1].Mu.RUnlock()
				defer cl.lst[i].Mu.RUnlock()
				return cl.lst[i-1]
			} else {
				cl.lst[i-1].Mu.RUnlock()
				cl.lst[i].Mu.RUnlock()
			}
		}

		if i == 0 {
			return cl.lst[0]
		}
	}
}

func (cl *ContainerList) GetLoadPercentage() (int, float64) {
	sum := uint32(0)
	cl.mu.RLock()
	for _, cnt := range cl.lst {
		sum += uint32(min(1, cnt.Fluxes/cnt.MaxFluxes))
	}

	length := len(cl.lst)

	cl.mu.RUnlock()

	return length, float64(sum) / float64(length)
}

func (cl *ContainerList) Size() int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return len(cl.lst)
}
