package infrastructure

import (
	"time"

	"github.com/redis/go-redis/v9"
)

func NewRedisBroker(addr string, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:        addr,
		Password:    password,
		DB:          db,
		ReadTimeout: 10 * time.Second,
	})
}