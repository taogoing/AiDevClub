package platform

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

func NewRateLimiter(rdb *redis.Client, limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{rdb: rdb, limit: limit, window: window}
}

func (l *RateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	n, err := l.rdb.Incr(ctx, key).Result()
	if err != nil {
		return false, err
	}
	if n == 1 {
		if err := l.rdb.Expire(ctx, key, l.window).Err(); err != nil {
			return false, err
		}
	}
	return n <= int64(l.limit), nil
}

func RateLimitMiddleware(rdb *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	limiter := NewRateLimiter(rdb, limit, window)
	return func(c *gin.Context) {
		key := fmt.Sprintf("ratelimit:%s:%s", c.FullPath(), c.ClientIP())
		allowed, err := limiter.Allow(c.Request.Context(), key)
		if err != nil {
			c.Next()
			return
		}
		if !allowed {
			Fail(c, http.StatusTooManyRequests, 42901, "请求过于频繁，请稍后再试")
			c.Abort()
			return
		}
		c.Next()
	}
}
