package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

// Window defines a rolling time window for analytics.
type Window struct {
	Name     string        // e.g. "1m", "5m", "1h", "24h"
	Duration time.Duration // actual duration
}

var windows = []Window{
	{Name: "1m", Duration: 1 * time.Minute},
	{Name: "5m", Duration: 5 * time.Minute},
	{Name: "1h", Duration: 1 * time.Hour},
	{Name: "24h", Duration: 24 * time.Hour},
}

// priceEntry stores a historical price point in memory.
type priceEntry struct {
	price     float64
	timestamp int64 // unix ms
}

// WindowTracker maintains in-memory rolling price windows and updates Redis sorted sets.
type WindowTracker struct {
	rdb *redis.Client

	mu      sync.Mutex
	history map[string][]priceEntry // symbol -> time-ordered price entries
}

func NewWindowTracker(rdb *redis.Client) *WindowTracker {
	return &WindowTracker{
		rdb:     rdb,
		history: make(map[string][]priceEntry),
	}
}

// Record adds a price event and updates Top-K sorted sets for all windows.
func (w *WindowTracker) Record(ctx context.Context, event types.PriceUpdateEvent) {
	w.mu.Lock()

	// Append to history
	w.history[event.Symbol] = append(w.history[event.Symbol], priceEntry{
		price:     event.Price,
		timestamp: event.Timestamp,
	})

	// Evict entries older than the largest window (24h)
	maxAge := event.Timestamp - int64(24*time.Hour/time.Millisecond)
	entries := w.history[event.Symbol]
	cutoff := 0
	for cutoff < len(entries) && entries[cutoff].timestamp < maxAge {
		cutoff++
	}
	if cutoff > 0 {
		w.history[event.Symbol] = entries[cutoff:]
	}

	// Snapshot current history for this symbol
	snapshot := make([]priceEntry, len(w.history[event.Symbol]))
	copy(snapshot, w.history[event.Symbol])

	w.mu.Unlock()

	// Calculate % drop for each window and update Redis
	for _, win := range windows {
		pctDrop := calcDrop(snapshot, event.Timestamp, win.Duration)
		if pctDrop == 0 {
			continue
		}

		key := fmt.Sprintf("top-drops:%s", win.Name)
		err := w.rdb.ZAdd(ctx, key, redis.Z{
			Score:  pctDrop,
			Member: event.Symbol,
		}).Err()
		if err != nil {
			log.Printf("[Analytics] Redis ZADD error (%s): %v", key, err)
			continue
		}

		w.rdb.Expire(ctx, key, win.Duration)
	}
}

// calcDrop computes the percentage price drop within a given window.
func calcDrop(entries []priceEntry, now int64, window time.Duration) float64 {
	windowStart := now - int64(window/time.Millisecond)

	var oldestPrice float64
	found := false
	for _, e := range entries {
		if e.timestamp >= windowStart {
			oldestPrice = e.price
			found = true
			break
		}
	}

	if !found || oldestPrice == 0 {
		return 0
	}

	latestPrice := entries[len(entries)-1].price
	pctChange := ((latestPrice - oldestPrice) / oldestPrice) * 100

	if pctChange >= 0 {
		return 0
	}
	return -pctChange
}
