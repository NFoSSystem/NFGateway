package utils

type ruleFunc *func([]byte) bool

type RuleMap map[ruleFunc]chan []byte

func (rm RuleMap) Add(f func([]byte) bool, pktChan chan []byte) {
	rf := ruleFunc(&f)
	rm[rf] = pktChan
}

func (rm RuleMap) GetChan(pkt []byte) chan []byte {
	for rule, cntChan := range rm {
		if (func([]byte) bool)(*rule)(pkt) {
			return cntChan
		}
	}

	return nil
}
