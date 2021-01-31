package cnt

import (
	op "faasrouter/openwhisk"
	"faasrouter/utils"
	"math"
	"time"

	"github.com/google/uuid"
)

func calculateScalingDelta(cl *ContainerList, upThr, downThr float64, upInc, downDec float64, min int) int {
	s, lp := cl.GetLoadPercentage()
	utils.RLogger.Printf("Calculated scaling delta - n° of actions %d - load percentage %f\n", s, lp)
	if lp >= upThr {
		return int(math.Ceil(float64(s) * upInc))
	} else if lp <= downThr {
		delta := -int(math.Floor(float64(s) * downDec))
		if (s + delta) > min {
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
	size     int
	stopChan chan<- int
}

func startDropFunction(action, hostname, auth string, size int, timeout time.Duration, redisIp string, redisPort int, stopChan <-chan int) {
	timeoutT := time.NewTicker(timeout)
	uId := uuid.New()

	for i := 0; i < size; i++ {
		err := op.CreateFunction(hostname, auth, action, redisIp, redisPort)
		if err != nil {
			utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
				action, err)
		} else {
			utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), action, hostname)
		}
	}

	for {
		select {
		case <-timeoutT.C:
			utils.RLogger.Printf("[%s] Creation of %d %s function\n", uId.String(), size, action)
			for i := 0; i < size; i++ {
				err := op.CreateFunction(hostname, auth, action, redisIp, redisPort)
				if err != nil {
					utils.RLogger.Printf("[%s] Error creating function on OpenWhisk at hostname %s for action %s: %s\n", uId.String(), hostname,
						action, err)
				} else {
					utils.RLogger.Printf("[%s] Function %s created on OpenWhisk at %s", uId.String(), hostname, action)
				}
			}
		case dec := <-stopChan:
			utils.RLogger.Printf("[%s] Received instruction to remove %d functions\n", uId.String(), dec)
			if size-int(math.Abs(float64(dec))) <= 0 {
				return
			} else {
				size -= dec
			}
		}
	}
}

func (p *Provisioner) InstantiateFunctions() {
	increment := make(chan int)
	go p.HandleScaling(p.Check, p.Cl, increment)

	var iSlice []Instance

	stopChan := make(chan int)
	iSlice = append(iSlice, Instance{int(p.Min), stopChan})
	go startDropFunction(p.Action, p.Hostname, p.Auth, int(p.Min), p.Timeout, p.RedisIp, p.RedisPort, stopChan)

	for {
		select {
		case delta := <-increment:
			if delta > 0 {
				stopChan = make(chan int)
				iSlice = append(iSlice, Instance{delta, stopChan})
				go startDropFunction(p.Action, p.Hostname, p.Auth, int(p.Min), p.Timeout, p.RedisIp, p.RedisPort, stopChan)
			} else if delta < 0 {
				for _, i := range iSlice {
					if i.size >= delta {
						i.stopChan <- delta
						i.size -= delta
						break
					} else {
						//  when i.size < delta
						i.stopChan <- i.size
						i.size = 0
						delta -= i.size
					}
				}
			}
		}
	}
}
