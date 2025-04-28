package distobject

import (
	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	client *redis.Client
}

func NewRedisClient(addr string) *RedisClient {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
		DB:   0,
	})
	return &RedisClient{client: rdb}
}

func (r *RedisClient) Client() *redis.Client {
	return r.client
}
