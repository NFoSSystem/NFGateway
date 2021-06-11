package initnf

import (
	"faasrouter/utils"

	"bitbucket.org/Manaphy91/faasnat/natdb"
)

const LOCAL_INTERFACE = "127.0.0.1"

func InitNat() {
	natSc := natdb.New("172.17.0.1", 6379)
	err := natSc.CleanUpSets()
	if err != nil {
		utils.RLogger.Fatalf("Error during NAT NF bitsets clean up: %s", err)
	}

	err = natSc.CleanUpFlowSets()
	if err != nil {
		utils.RLogger.Fatalf("Error during NAT NF in/out flows clean up: %s", err)
	}

	err = natSc.InitBitSets()
	if err != nil {
		utils.RLogger.Fatalf("Error during NAT NF bitsets init: %s", err)
	}
}
