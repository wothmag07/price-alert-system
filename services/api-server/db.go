package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewDB(ctx context.Context, url string) *pgxpool.Pool {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		log.Fatalf("[DB] Failed to connect: %v", err)
	}
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[DB] Ping failed: %v", err)
	}
	log.Println("[DB] Connected to PostgreSQL")
	return pool
}

func RunMigrations(ctx context.Context, pool *pgxpool.Pool) {
	schema := `
	CREATE TABLE IF NOT EXISTS users (
		id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		email       VARCHAR(255) UNIQUE NOT NULL,
		password    VARCHAR(255) NOT NULL,
		created_at  TIMESTAMP DEFAULT NOW(),
		updated_at  TIMESTAMP DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS alerts (
		id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
		symbol        VARCHAR(20) NOT NULL,
		condition     VARCHAR(30) NOT NULL,
		threshold     DECIMAL(20, 8) NOT NULL,
		status        VARCHAR(20) DEFAULT 'ACTIVE',
		created_at    TIMESTAMP DEFAULT NOW(),
		triggered_at  TIMESTAMP
	);

	CREATE INDEX IF NOT EXISTS idx_alerts_symbol_status ON alerts(symbol, status);
	CREATE INDEX IF NOT EXISTS idx_alerts_user_id ON alerts(user_id);

	CREATE TABLE IF NOT EXISTS alert_history (
		id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
		alert_id        UUID NOT NULL REFERENCES alerts(id) ON DELETE CASCADE,
		triggered_price DECIMAL(20, 8) NOT NULL,
		notification    JSONB,
		created_at      TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_alert_history_alert_id ON alert_history(alert_id);

	CREATE TABLE IF NOT EXISTS price_history (
		id        BIGSERIAL PRIMARY KEY,
		symbol    VARCHAR(20) NOT NULL,
		price     DECIMAL(20, 8) NOT NULL,
		volume    DECIMAL(20, 8),
		timestamp TIMESTAMP NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_price_history_symbol_ts ON price_history(symbol, timestamp DESC);
	`

	if _, err := pool.Exec(ctx, schema); err != nil {
		log.Fatalf("[DB] Migration failed: %v", err)
	}
	fmt.Println("[DB] Migrations complete")
}
