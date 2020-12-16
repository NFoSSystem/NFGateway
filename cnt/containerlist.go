package cnt

import (
	"sync"
)

type ContainerList struct {
	lst      []*container
	mu       sync.RWMutex
	duration uint64
}

func NewContainerList() *ContainerList {
	cl := new(ContainerList)
	cl.mu = sync.RWMutex{}
	return cl
}

func (cl *ContainerList) AddContainer(cnt *container) {
	cl.mu.Lock()
	cl.lst = append(cl.lst, cnt)
	cl.mu.Unlock()
}

func (cl *ContainerList) GetList() []*container {
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

func (cl *ContainerList) GetContainer() *container {
	for {
		if cl.Empty() {
			continue
		}

		cl.mu.RLock()
		defer cl.mu.RUnlock()
		length := len(cl.lst)
		var i int
		for i = length - 1; i > 0; i-- {
			cl.lst[i].mu.RLock()
			cl.lst[i-1].mu.RLock()
			fMin := min(cl.lst[i].fluxes, cl.lst[i-1].fluxes)
			if fMin != cl.lst[i-1].fluxes {
				defer cl.lst[i-1].mu.RUnlock()
				defer cl.lst[i].mu.RUnlock()
				return cl.lst[i-1]
			} else {
				cl.lst[i-1].mu.RUnlock()
				cl.lst[i].mu.RUnlock()
			}
		}

		if i == 0 {
			return cl.lst[0]
		}
	}
}

func (cl *ContainerList) GetLoadPercentage() (int, float64) {
	sum := int8(0)
	cl.mu.RLock()
	for _, cnt := range cl.lst {
		sum += min(1, cnt.fluxes)
	}

	length := len(cl.lst)

	defer cl.mu.RLock()
	return length, float64(sum) / float64(length)
}

func (cl *ContainerList) Size() int {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return len(cl.lst)
}
