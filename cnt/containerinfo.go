package cnt

type ContainerInfo struct {
	cntChan chan<- *container
	cntLst  *ContainerList
}
