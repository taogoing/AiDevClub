package platform

import "github.com/redis/go-redis/v9"

func OpenRedis(addr, password string, db int) *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     20,
		MinIdleConns: 5,
		PoolTimeout:  30,
	})
}
