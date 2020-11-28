package utils

import "log"

var RLogger *log.Logger

func init() {
	var err error
	RLogger, err = NewRedisLogger(LOGGER_PREFIX, LOGGER_PUB_SUB_CHAN, LOCAL_INTERFACE, 6379)
	if err != nil {
		log.Fatal(err)
	}
}
