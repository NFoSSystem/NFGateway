package cnt

import (
	"faasrouter/utils"
)

type ContainerInfo struct {
	cntPool *utils.ConnPool
	cntLst  *ContainerList
	maxFluxes int8
}
