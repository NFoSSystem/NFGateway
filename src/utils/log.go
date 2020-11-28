package utils

import (
	"fmt"
	"log"
	"time"

	"github.com/piaohao/godis"
)

type RedisPubSubWriter struct {
	chanName string
	client   *godis.Redis
}

func NewRedisPubSubWriter(chanName string, hostname string, port uint16) (*RedisPubSubWriter, error) {
	opt := &godis.Option{
		Host: hostname,
		Port: int(port),
		Db:   0,
	}

	pool := godis.NewPool(&godis.PoolConfig{}, opt)
	if pool == nil {
		return nil, fmt.Errorf("Error opening Redis Pub/Sub pool!")
	}

	res := new(RedisPubSubWriter)
	res.chanName = chanName

	cli, err := pool.GetResource()
	if err != nil {
		return nil, fmt.Errorf("Error opening Redis Pub/Sub pool!")
	}
	res.client = cli

	return res, nil
}

func (r *RedisPubSubWriter) Write(buff []byte) (int, error) {
	if r.client == nil || r.chanName == "" {
		log.Println(string(buff))
	}
	r.client.Publish(r.chanName, string(buff))
	return len(buff), nil
}

func (r *RedisPubSubWriter) Close() error {
	return r.client.Close()
}

func GetNanoSeconds() int64 {
	t := time.Now()
	return t.UnixNano()
}
