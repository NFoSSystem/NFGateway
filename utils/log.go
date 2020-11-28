package utils

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisPubSubWriter struct {
	client  *redis.Client
	publish func(buff []byte) (int, error)
}

func NewRedisPubSubWriter(chanName string, hostname string, port uint16) (*RedisPubSubWriter, error) {
	ctx := context.Background()

	res := new(RedisPubSubWriter)
	res.client = redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    hostname + ":" + strconv.Itoa(int(port)),
	})

	pSClient := res.client.Subscribe(ctx, chanName)
	_, err := pSClient.Receive(ctx)

	res.publish = func(buff []byte) (int, error) {
		size, err := res.client.Publish(ctx, chanName, string(buff)).Result()
		return int(size), err
	}

	return res, fmt.Errorf("Error attempt opening Redis Pub/Sub Channel: %s",
		err)
}

func (r *RedisPubSubWriter) Write(buff []byte) (int, error) {
	return r.publish(buff)
}

func (r *RedisPubSubWriter) Close() error {
	return r.client.Close()
}

func GetNanoSeconds() int64 {
	t := time.Now()
	return t.UnixNano()
}
