package utils

import (
	"fmt"
	"net"
)

func FastPow(base, exp int) int {
	res := base
	for i := exp; i > 0; i-- {
		res *= base
	}
	return res
}

func GetPortsFromBytes(ptr1, ptr2, ptr3, ptr4 int) func(packet []byte) (uint16, uint16, error) {
	return func(packet []byte) (uint16, uint16, error) {
		if len(packet) < 4 {
			return 0, 0, fmt.Errorf("Error provided byte slice with lenght lower than 4")
		}

		var source uint16 = (uint16(packet[ptr1]|0) << 8) | uint16(packet[ptr2])
		var target uint16 = (uint16(packet[ptr3]|0) << 8) | uint16(packet[ptr4])

		return source, target, nil
	}
}

func max(v1, v2 int) int {
	if v1 > v2 {
		return v1
	} else {
		return v2
	}
}

func fastMax(values ...int) int {
	if values == nil || len(values) == 0 {
		return -1
	} else {
		if len(values) == 1 {
			return values[0]
		}

		var vMax = values[0]
		for val := range values[1:] {
			vMax = max(vMax, val)
		}
		return vMax
	}
}

func GetIPsFromBytes(ptr1, ptr2 int) func([]byte) (net.IP, net.IP, error) {
	return func(pkt []byte) (net.IP, net.IP, error) {
		if len(pkt) < max(ptr1+3, ptr2+3) {
			return nil, nil, fmt.Errorf("Error byte buffer provided smaller than expected!")
		}
		return net.IPv4(pkt[ptr1], pkt[ptr1+1], pkt[ptr1+2], pkt[ptr1+3]),
			net.IPv4(pkt[ptr2], pkt[ptr2+1], pkt[ptr2+2], pkt[ptr2+3]),
			nil
	}
}

func Memset(slice []byte, val byte) {
	if len(slice) == 0 {
		return
	}

	slice[0] = val
	for i := 1; i < len(slice); i *= 2 {
		copy(slice[i:], slice[:i])
	}
}

type Buffer struct {
	buff      []byte
	cleanChan chan<- *Buffer
	id        uint8
}

func (b *Buffer) Buff() []byte {
	return b.buff
}

func (b *Buffer) Release() {
	b.cleanChan <- b
}

/*
BuffersPool consists of a fixed length pool of byte buffers with a given size
*/
type BuffersPool struct {
	pool      chan *Buffer
	cleanChan <-chan *Buffer
	buffLst   []*Buffer
}

func cleanUpRoutine(index int, cleanChan <-chan *Buffer, pool chan *Buffer) {
	for {
		b := <-cleanChan
		Memset(b.buff, 0)
		pool <- b
	}
}

/*
NewBuffersPool takes as arguments the buffSize and the poolSize, respectively size of each buffer and number of
byte buffers into the pool
*/
func NewBuffersPool(buffSize uint16, poolSize uint8) *BuffersPool {
	bp := new(BuffersPool)
	bp.pool = make(chan *Buffer, poolSize)
	cleanChan := make(chan *Buffer, poolSize)
	bp.cleanChan = cleanChan

	for i := uint8(0); i < uint8(poolSize/10); i++ {
		go cleanUpRoutine(int(i), bp.cleanChan, bp.pool)
	}

	for i := uint8(0); i < poolSize; i++ {
		b := &Buffer{}
		b.buff = make([]byte, buffSize)
		b.cleanChan = cleanChan
		b.id = i
		bp.buffLst = append(bp.buffLst, b)

		bp.pool <- b
	}

	return bp
}

/*
Next method returns the next available buffer of a given BuffersPool
*/
func (b *BuffersPool) Next() *Buffer {
	buffer := <-b.pool
	return buffer
}
