package cnt

import (
	op "faasrouter/openwhisk"
	"faasrouter/utils"
	"log"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

func calculateScalingDelta(cl *ContainerList, upThr, downThr float64, upInc, downDec float64, min int) int {
	s, lp := cl.GetLoadPercentage()
	utils.RLogger.Printf("Calculated scaling delta - n° of actions %d - load percentage %f\n", s, lp)
	if lp >= upThr {
		return int(math.Ceil(float64(s) * upInc))
	} else if lp <= downThr {
		delta := -int(math.Max(math.Floor(float64(s)*downDec), 1.0))
		if (s + delta) >= min {
			return delta
		}
		return 0
	} else {
		return 0
	}
}

type Provisioner struct {
	Action    string
	Hostname  string
	Auth      string
	Timeout   time.Duration
	Check     time.Duration
	Cl        *ContainerList
	UpThr     float64
	DownThr   float64
	UpInc     float64
	DownInc   float64
	Min       int
	RedisIp   string
	RedisPort int
}

func (p *Provisioner) HandleScaling(check time.Duration, cl *ContainerList, increment chan<- int) {
	checkT := time.NewTicker(check)

	for {
		select {
		case <-checkT.C:
			res := calculateScalingDelta(cl, p.UpThr, p.DownThr, p.UpInc, p.DownInc, p.Min)
			increment <- res
		}
	}
}

type Instance struct {
	size    int
	decChan chan uint16
}

// func startDropFunction(action, hostname, auth string, size int, timeout time.Duration, redisIp string, redisPort int, decChan <-chan int, stopChan <-chan struct{}) {
// 	timeoutT := time.NewTicker(timeout)
// 	uId := uuid.New()

// 	for i := 0; i < size; i++ {
// 		err := op.CreateFunction(hostname, auth, action, redisIp, redisPort)
// 		if err != nil {
// 			utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
// 				action, err)
// 		} else {
// 			utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), action, hostname)
// 		}
// 	}

// 	for {
// 		select {
// 		case <-stopChan:
// 			return
// 		case <-timeoutT.C:
// 			utils.RLogger.Printf("[%s] Creation of %d %s function\n", uId.String(), size, action)
// 			for i := 0; i < size; i++ {
// 				err := op.CreateFunction(hostname, auth, action, redisIp, redisPort)
// 				if err != nil {
// 					utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
// 						action, err)
// 				} else {
// 					utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), hostname, action)
// 				}
// 			}
// 		case dec := <-decChan:
// 			log.Printf("DEBUG startDropFunction dec -> %d", dec)
// 			utils.RLogger.Printf("[%s] Received instruction to remove %d functions\n", uId.String(), dec)
// 			if size+dec <= 0 {
// 				// drain the chan
// 				for _ = range stopChan {

// 				}
// 				return
// 			} else {
// 				size += dec
// 			}
// 		}
// 	}
// }

type Action2Cep struct {
	mu  sync.RWMutex
	a2c map[string]*ConnElemProvisioner
}

func NewAction2Cep() *Action2Cep {
	res := new(Action2Cep)
	res.a2c = make(map[string]*ConnElemProvisioner)
	return res
}

func (a *Action2Cep) Set(action string, cep *ConnElemProvisioner) {
	a.mu.Lock()
	a.a2c[action] = cep
	a.mu.Unlock()
}

func (a *Action2Cep) Get(action string) *ConnElemProvisioner {
	a.mu.RLock()
	res := a.a2c[action]
	a.mu.RUnlock()
	return res
}

type ConnElemProvisioner struct {
	c2e map[uint16]*utils.ConnPoolElem
	c2l map[uint16]*ContainerList
	c2p map[uint16]*Instance
	mu  sync.RWMutex
}

func NewConnElemProvisioner() *ConnElemProvisioner {
	res := new(ConnElemProvisioner)
	res.c2e = make(map[uint16]*utils.ConnPoolElem)
	res.c2l = make(map[uint16]*ContainerList)
	res.c2p = make(map[uint16]*Instance)
	return res
}

func (c *ConnElemProvisioner) AddInstance(cntId uint16, inst *Instance) {
	c.mu.Lock()
	c.c2p[cntId] = inst
	c.mu.Unlock()
}

func (c *ConnElemProvisioner) GetInstance(cntId uint16) *Instance {
	c.mu.RLock()
	res, _ := c.c2p[cntId]
	c.mu.RUnlock()
	return res
}

func (c *ConnElemProvisioner) GetInstanceNoLock(cntId uint16) *Instance {
	res, _ := c.c2p[cntId]
	return res
}

func (c *ConnElemProvisioner) RemoveInstance(cntId uint16) {
	c.mu.Lock()
	delete(c.c2p, cntId)
	c.mu.Unlock()
}

func (c *ConnElemProvisioner) AddConnElem(cntId uint16, ce *utils.ConnPoolElem, cl *ContainerList) {
	c.mu.Lock()
	c.c2e[cntId] = ce
	c.c2l[cntId] = cl
	c.mu.Unlock()
}

func (c *ConnElemProvisioner) GetConnElem(cntId uint16) (*utils.ConnPoolElem, *ContainerList) {
	c.mu.RLock()
	ce, _ := c.c2e[cntId]
	cl, _ := c.c2l[cntId]
	c.mu.RUnlock()
	return ce, cl
}

func (c *ConnElemProvisioner) RemoveConnElem(cntId uint16) {
	c.mu.Lock()
	delete(c.c2e, cntId)
	c.mu.Unlock()
}

func startDropFunctionInit(action, hostname, auth string, timeout time.Duration, redisIp string, redisPort int, cep *ConnElemProvisioner, inst *Instance) {
	timeoutT := time.NewTicker(timeout)
	uId := uuid.New()

	min := inst.size

	var cntIdMap map[uint16]bool = make(map[uint16]bool)

	for {
		select {
		case <-timeoutT.C:
			utils.RLogger.Printf("[%s] Creation of %d %s function\n", uId.String(), min, action)
			for i := 0; i < min; i++ {
				cntId := uint16(rand.Intn(65535)) + 1
				err := op.CreateFunction(hostname, auth, action, redisIp, redisPort, cntId, "1")
				cntIdMap[cntId] = true
				cep.AddInstance(cntId, inst)
				if err != nil {
					utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
						action, err)
				} else {
					utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), hostname, action)
				}
			}
		}
	}
}

func startDropFunction(action, hostname, auth string, timeout time.Duration, redisIp string, redisPort int, cep *ConnElemProvisioner, inst *Instance) {
	timeoutT := time.NewTicker(timeout)
	uId := uuid.New()

	var cntIdMap map[uint16]bool = make(map[uint16]bool, 100)

	for i := 0; i < inst.size; i++ {
		cntId := uint16(rand.Intn(65535)) + 1
		err := op.CreateFunction(hostname, auth, action, redisIp, redisPort, cntId, "0")
		cntIdMap[cntId] = true
		cep.AddInstance(cntId, inst)
		if err != nil {
			utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
				action, err)
		} else {
			utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), action, hostname)
		}
	}

	for {
		utils.RLogger.Printf("[%s] Create / delete loop enter\n", uId.String())
		select {
		case <-timeoutT.C:
			if inst.size <= 0 {
				return
			}

			utils.RLogger.Printf("[%s] Creation of %d %s function\n", uId.String(), inst.size, action)
			for i := 0; i < inst.size; i++ {
				cntId := uint16(rand.Intn(65535)) + 1
				err := op.CreateFunction(hostname, auth, action, redisIp, redisPort, cntId, "0")
				cntIdMap[cntId] = true
				cep.AddInstance(cntId, inst)
				if err != nil {
					utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
						action, err)
				} else {
					utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), hostname, action)
				}
				if inst.size <= 0 {
					return
				}
			}
		case cntInt := <-inst.decChan:
			utils.RLogger.Printf("[%s] Received instruction to remove function with cntId %d\n", uId.String(), cntInt)
			if _, ok := cntIdMap[cntInt]; ok {
				utils.RLogger.Printf("[%s] Before delete cntId from cntIdMap - cntId %d", uId.String(), cntInt)
				delete(cntIdMap, cntInt)
				utils.RLogger.Printf("[%s] Before remove container from cep - cntId %d", uId.String(), cntInt)
				cep.RemoveInstance(cntInt)
				utils.RLogger.Printf("[%s] After delete of container %d", uId.String(), cntInt)
				connElem, cntLst := cep.GetConnElem(cntInt)
				container := connElem.GetContainer()
				connElem.Delete()
				utils.RLogger.Printf("[%s] After delete of connElem %d", uId.String(), cntInt)
				cntLst.RemoveContainer(container)
				utils.RLogger.Printf("[%s] After removing cntId %d from cntLst", uId.String(), cntInt)
				utils.RLogger.Printf("[%s] Before deleting one from inst.size %d", uId.String(), inst.size)
				inst.size -= 1
				utils.RLogger.Printf("[%s] Delete function %d exit", uId.String(), cntInt)
			} else {
				utils.RLogger.Printf("[%s] Function with cntId %d not anymore present\n", uId.String(), cntInt)
			}
		}
	}
}

// func (p *Provisioner) InstantiateFunctions(c2e map[uint16]*utils.ConnPoolElem) {
// 	increment := make(chan int)
// 	go p.HandleScaling(p.Check, p.Cl, increment)

// 	var iSlice []Instance

// 	decChan := make(chan int)
// 	stopChan := make(chan struct{})
// 	iSlice = append(iSlice, Instance{int(p.Min), decChan, stopChan})
// 	go startDropFunction(p.Action, p.Hostname, p.Auth, int(p.Min), p.Timeout, p.RedisIp, p.RedisPort, decChan, stopChan)

// 	for {
// 		log.Println("InstatiateFunctions loop")
// 		select {
// 		case delta := <-increment:
// 			log.Printf("Received delta: %d", delta)
// 			if delta > 0 {
// 				decChan := make(chan int)
// 				stopChan := make(chan struct{})
// 				iSlice = append(iSlice, Instance{delta, decChan, stopChan})
// 				go startDropFunction(p.Action, p.Hostname, p.Auth, int(p.Min), p.Timeout, p.RedisIp, p.RedisPort, decChan, stopChan)
// 			} else if delta < 0 {
// 				for _, i := range iSlice {
// 					if i.size >= -1*delta {
// 						log.Printf("DEBUG 1 i.size %d - delta %d", i.size, delta)
// 						i.stopChan <- delta
// 						log.Printf("DEBUG 1 After")
// 						i.size -= delta
// 						break
// 					} else {
// 						log.Println("DEBUG 2 i.size %d - delta %d", i.size, delta)
// 						//  when i.size < delta
// 						i.stopChan <- (-1 * i.size)
// 						log.Println("DEBUG 2 After")
// 						i.size = 0
// 						delta -= i.size
// 					}
// 				}
// 			}
// 		}
// 	}
// }

func (p *Provisioner) InstantiateFunctions(cep *ConnElemProvisioner) {
	increment := make(chan int)
	go p.HandleScaling(p.Check, p.Cl, increment)

	inst := &Instance{p.Min, make(chan uint16)}
	tick := time.NewTicker(p.Timeout)
	//go startDropFunctionInit(p.Action, p.Hostname, p.Auth, p.Timeout, p.RedisIp, p.RedisPort, cep, inst)

	go func() {
		for i := 0; i < p.Min; i++ {
			cntId := uint16(rand.Intn(65535)) + 1
			err := op.CreateFunction(p.Hostname, p.Auth, p.Action, p.RedisIp, p.RedisPort, cntId, "1")
			cep.AddInstance(cntId, inst)
			if err != nil {
				utils.RLogger.Printf("Error creating function on OpenWhisk at hostname %s for action %s: %s\n", p.Action)
			} else {
				utils.RLogger.Printf("Function %s created on OpenWhisk", p.Action)
			}
		}

		for {
			<-tick.C
			utils.RLogger.Printf("Creation of %d %s function\n", p.Min, p.Action)
			for i := 0; i < p.Min; i++ {
				cntId := uint16(rand.Intn(65535)) + 1
				err := op.CreateFunction(p.Hostname, p.Auth, p.Action, p.RedisIp, p.RedisPort, cntId, "1")
				utils.RLogger.Printf("Container created with id %d\n", cntId)
				cep.AddInstance(cntId, inst)
				if err != nil {
					utils.RLogger.Printf("Error creating function on OpenWhisk at hostname %s for action %s: %s\n", p.Action)
				} else {
					utils.RLogger.Printf("Function %s created on OpenWhisk", p.Action)
				}
			}
		}
	}()

	for {
		select {
		// case <-tick.C:
		// 	utils.RLogger.Printf("Creation of %d %s function\n", p.Min, p.Action)
		// 	for i := 0; i < p.Min; i++ {
		// 		cntId := uint16(rand.Intn(65535)) + 1
		// 		err := op.CreateFunction(p.Hostname, p.Auth, p.Action, p.RedisIp, p.RedisPort, cntId, "1")
		// 		utils.RLogger.Printf("Container created with id %d\n", cntId)
		// 		cep.AddInstance(cntId, inst)
		// 		if err != nil {
		// 			utils.RLogger.Printf("Error creating function on OpenWhisk at hostname %s for action %s: %s\n", p.Action)
		// 		} else {
		// 			utils.RLogger.Printf("Function %s created on OpenWhisk", p.Action)
		// 		}
		// 	}
		case delta := <-increment:
			if delta > 0 {
				inst := &Instance{delta, make(chan uint16, delta+1)}
				go startDropFunction(p.Action, p.Hostname, p.Auth, p.Timeout, p.RedisIp, p.RedisPort, cep, inst)
			} else if delta < 0 {
				cep.mu.RLock()
				for cntId, connElem := range cep.c2e {
					if connElem.IsAlive() && !connElem.IsRepl() {
						inst := cep.GetInstanceNoLock(cntId)
						log.Printf("DEBUG Before sending %d to decChan - connElem.IsAlive() %v - connElem.IsRepl() %v\n", cntId, connElem.IsAlive(),
							connElem.IsRepl())
						if inst == nil || inst.decChan == nil {
							delete(cep.c2e, cntId)
							continue
						}
						inst.decChan <- cntId
						log.Printf("DEBUG After sending %d to decChan - connElem.IsAlive() %v - connElem.IsRepl() %v\n", cntId, connElem.IsAlive(),
							connElem.IsRepl())
						delta += 1

						if delta == 0 {
							break
						}
					} else if !connElem.IsAlive() {
						delete(cep.c2e, cntId)
					}
				}
				cep.mu.RUnlock()
			}
		}
	}
}
