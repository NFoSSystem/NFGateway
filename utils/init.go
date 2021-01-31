package utils

import (
	"faasrouter/nat"
	"log"
	"time"
)

var RLogger *log.Logger

var CPMap *nat.CntPortMapping
var PCMap *nat.PortCntMapping

func init() {
	var err error
	RLogger, err = NewRedisLogger(LOGGER_PREFIX, LOGGER_PUB_SUB_CHAN, LOCAL_INTERFACE, 6379)
	if err != nil {
		log.Fatal(err)
	}

	CPMap = nat.NewCntPortMapping()
	PCMap = nat.NewPortCntMapping(120 * time.Second)
}
