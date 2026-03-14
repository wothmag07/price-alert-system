package middleware

import (
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type RateLimiterConfig struct {
	WindowMs int
	Max      int
}

func RateLimiter(rdb *redis.Client, config *RateLimiterConfig) gin.HandlerFunc {
	authConfig := RateLimiterConfig{WindowMs: 60_000, Max: 100}
	unauthConfig := RateLimiterConfig{WindowMs: 60_000, Max: 20}
	if config != nil {
		authConfig = *config
		unauthConfig = *config
	}

	return func(c *gin.Context) {
		ctx := c.Request.Context()

		userID, _ := c.Get("userId")
		cfg := unauthConfig
		var key string
		if userID != nil {
			cfg = authConfig
			key = fmt.Sprintf("rate-limit:%s:api", userID)
		} else {
			key = fmt.Sprintf("rate-limit:%s:api", c.ClientIP())
		}

		now := time.Now().UnixMilli()
		windowStart := now - int64(cfg.WindowMs)
		member := fmt.Sprintf("%d-%f", now, rand.Float64())

		pipe := rdb.Pipeline()
		pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
		pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member})
		countCmd := pipe.ZCard(ctx, key)
		pipe.PExpire(ctx, key, time.Duration(cfg.WindowMs)*time.Millisecond)
		_, err := pipe.Exec(ctx)

		if err != nil {
			// On Redis failure, allow the request through
			c.Next()
			return
		}

		count := countCmd.Val()
		c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Max))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(max(0, cfg.Max-int(count))))

		if int(count) > cfg.Max {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			return
		}

		c.Next()
	}
}

func AlertCreationLimiter(rdb *redis.Client) gin.HandlerFunc {
	return RateLimiter(rdb, &RateLimiterConfig{WindowMs: 3_600_000, Max: 10})
}
