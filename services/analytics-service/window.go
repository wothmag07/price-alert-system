package main

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
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

// PriceUpdateEvent mirrors the event from price-ingestion.
type PriceUpdateEvent struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Change24h float64 `json:"change24h"`
	Timestamp int64   `json:"timestamp"`
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
func (w *WindowTracker) Record(ctx context.Context, event PriceUpdateEvent) {
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
		// Score is the absolute percentage drop (higher = bigger drop)
		err := w.rdb.ZAdd(ctx, key, redis.Z{
			Score:  pctDrop,
			Member: event.Symbol,
		}).Err()
		if err != nil {
			log.Printf("[Analytics] Redis ZADD error (%s): %v", key, err)
			continue
		}

		// Set TTL matching the window duration
		w.rdb.Expire(ctx, key, win.Duration)
	}
}

// calcDrop computes the percentage price drop within a given window.
// Returns a positive number for drops (e.g. 5.2 means price dropped 5.2%).
// Returns 0 if no meaningful calculation can be made.
func calcDrop(entries []priceEntry, now int64, window time.Duration) float64 {
	windowStart := now - int64(window/time.Millisecond)

	// Find the earliest price within the window
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

	// Latest price is the last entry
	latestPrice := entries[len(entries)-1].price

	// Calculate % change (negative = drop)
	pctChange := ((latestPrice - oldestPrice) / oldestPrice) * 100

	// We only care about drops, so return the absolute drop value
	if pctChange >= 0 {
		return 0
	}
	return -pctChange // make positive
}
