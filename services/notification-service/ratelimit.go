package main

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const rateLimitWindow = time.Hour

// checkRateLimit returns true if the user is within their notification rate limit.
func checkRateLimit(ctx context.Context, rdb *redis.Client, userID string, maxPerHour int) (bool, error) {
	key := fmt.Sprintf("rate-limit:%s:notifications", userID)
	now := time.Now().UnixMilli()
	windowStart := now - int64(rateLimitWindow/time.Millisecond)
	member := fmt.Sprintf("%d-%f", now, rand.Float64())

	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(windowStart, 10))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: member})
	countCmd := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, rateLimitWindow)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	return countCmd.Val() <= int64(maxPerHour), nil
}
