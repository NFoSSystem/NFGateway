package cnt

import (
	"nfgateway/utils"
)

type ContainerInfo struct {
	cntPool   *utils.ConnPool
	cntLst    *ContainerList
	maxFluxes int8
}
