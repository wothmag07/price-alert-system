package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	alertCacheKeyFmt = "alerts:active:%s"
	alertCacheTTL    = 5 * time.Minute
)

// AlertStore loads active alerts with Redis caching and PostgreSQL fallback.
type AlertStore struct {
	db  *pgxpool.Pool
	rdb *redis.Client
}

func NewAlertStore(db *pgxpool.Pool, rdb *redis.Client) *AlertStore {
	return &AlertStore{db: db, rdb: rdb}
}

// GetActiveAlerts returns active alerts for a symbol.
// Checks Redis cache first, falls back to PostgreSQL on cache miss.
func (s *AlertStore) GetActiveAlerts(ctx context.Context, symbol string) ([]AlertRule, error) {
	cacheKey := fmt.Sprintf(alertCacheKeyFmt, symbol)

	// Try Redis cache
	cached, err := s.rdb.SMembers(ctx, cacheKey).Result()
	if err == nil && len(cached) > 0 {
		rules := make([]AlertRule, 0, len(cached))
		for _, raw := range cached {
			var rule AlertRule
			if err := json.Unmarshal([]byte(raw), &rule); err != nil {
				continue
			}
			rules = append(rules, rule)
		}
		return rules, nil
	}

	// Cache miss — query PostgreSQL
	rows, err := s.db.Query(ctx,
		`SELECT id, user_id, symbol, condition, threshold
		 FROM alerts
		 WHERE symbol = $1 AND status = 'ACTIVE'`,
		symbol,
	)
	if err != nil {
		return nil, fmt.Errorf("query alerts: %w", err)
	}
	defer rows.Close()

	rules := make([]AlertRule, 0)
	members := make([]interface{}, 0)

	for rows.Next() {
		var r AlertRule
		if err := rows.Scan(&r.ID, &r.UserID, &r.Symbol, &r.Condition, &r.Threshold); err != nil {
			log.Printf("[AlertStore] Scan error: %v", err)
			continue
		}
		rules = append(rules, r)
		jsonBytes, _ := json.Marshal(r)
		members = append(members, string(jsonBytes))
	}

	// Populate cache (even if empty — use short TTL to avoid repeated queries)
	if len(members) > 0 {
		pipe := s.rdb.Pipeline()
		pipe.SAdd(ctx, cacheKey, members...)
		pipe.Expire(ctx, cacheKey, alertCacheTTL)
		pipe.Exec(ctx)
	}

	return rules, nil
}

// MarkTriggered updates an alert's status to TRIGGERED in PostgreSQL and invalidates the cache.
func (s *AlertStore) MarkTriggered(ctx context.Context, alertID, symbol string, triggeredPrice float64) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx,
		`UPDATE alerts SET status = 'TRIGGERED', triggered_at = NOW() WHERE id = $1`,
		alertID,
	)
	if err != nil {
		return fmt.Errorf("update alert: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO alert_history (alert_id, triggered_price) VALUES ($1, $2)`,
		alertID, triggeredPrice,
	)
	if err != nil {
		return fmt.Errorf("insert history: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit: %w", err)
	}

	// Invalidate cache so next lookup fetches fresh data
	s.rdb.Del(ctx, fmt.Sprintf(alertCacheKeyFmt, symbol))

	return nil
}
