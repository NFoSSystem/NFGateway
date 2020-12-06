package cnt

import "sync"

type ContainerList struct {
	lst []*container
	mu  sync.Mutex
}

func NewContainerList() *ContainerList {
	cl := new(ContainerList)
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

func (c *ContainerList) Empty() bool {
	return len(c.lst) == 0
}
