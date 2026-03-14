package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

type AnalyticsHandler struct {
	rdb *redis.Client
}

func NewAnalyticsHandler(rdb *redis.Client) *AnalyticsHandler {
	return &AnalyticsHandler{rdb: rdb}
}

// TopDrops returns the Top-K biggest price drops for a given rolling window.
func (h *AnalyticsHandler) TopDrops(c *gin.Context) {
	ctx := c.Request.Context()

	window := c.DefaultQuery("window", "1h")
	switch window {
	case "1m", "5m", "1h", "24h":
		// valid
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid window. Use: 1m, 5m, 1h, 24h"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	key := fmt.Sprintf("top-drops:%s", window)

	// ZREVRANGE returns highest scores first (biggest drops)
	results, err := h.rdb.ZRevRangeWithScores(ctx, key, 0, int64(limit-1)).Result()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error"})
		return
	}

	type dropEntry struct {
		Symbol  string  `json:"symbol"`
		DropPct float64 `json:"dropPct"`
	}

	drops := make([]dropEntry, 0, len(results))
	for _, z := range results {
		drops = append(drops, dropEntry{
			Symbol:  z.Member.(string),
			DropPct: z.Score,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"window": window,
		"drops":  drops,
	})
}
