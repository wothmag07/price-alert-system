package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("[Notification] Service starting...")

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("[Notification] Postgres connect failed: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[Notification] Postgres ping failed: %v", err)
	}
	log.Println("[Notification] Connected to PostgreSQL")

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Notification] Redis connect failed: %v", err)
	}
	defer rdb.Close()
	log.Println("[Notification] Connected to Redis")

	// Notifier
	notifier := NewNotifier(cfg, pool, rdb)
	defer notifier.Close()

	log.Println("[Notification] Running...")
	notifier.Run(ctx)

	log.Println("[Notification] Shutdown complete")
}
