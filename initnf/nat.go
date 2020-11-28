package initnf

import (
	"faasrouter/utils"
	"log"

	"bitbucket.org/Manaphy91/faasnat/natdb"
)

const LOCAL_INTERFACE = "127.0.0.1"

func init() {
	rpsw, err := utils.NewRedisPubSubWriter("logChan", LOCAL_INTERFACE, 6379)
	if err != nil {
		log.Println("Error creating RedisPubSubWriter: %s", rpsw)
	}

	rLogger := log.New(rpsw, "[Gateway]", log.Ldate|log.Lmicroseconds)

	natSc := natdb.New(LOCAL_INTERFACE, 6379)
	err = natSc.CleanUpSets()
	if err != nil {
		rLogger.Fatalf("Error during NAT NF bitsets clean up: %s", err)
	}

	err = natSc.CleanUpFlowSets()
	if err != nil {
		rLogger.Fatalf("Error during NAT NF in/out flows clean up: %s", err)
	}

	err = natSc.InitBitSets()
	if err != nil {
		rLogger.Fatalf("Error during NAT NF bitsets init: %s", err)
	}
}
