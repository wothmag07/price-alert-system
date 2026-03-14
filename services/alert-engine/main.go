package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func main() {
	log.Println("[Alert Engine] Service starting...")

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		log.Fatalf("[Alert Engine] Postgres connect failed: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("[Alert Engine] Postgres ping failed: %v", err)
	}
	fmt.Println("[Alert Engine] Connected to PostgreSQL")

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Alert Engine] Redis connect failed: %v", err)
	}
	defer rdb.Close()
	fmt.Println("[Alert Engine] Connected to Redis")

	// Alert store + engine
	store := NewAlertStore(pool, rdb)
	engine := NewEngine(cfg.KafkaBrokers, store)
	defer engine.Close()

	fmt.Println("[Alert Engine] Running...")
	engine.Run(ctx)

	log.Println("[Alert Engine] Shutdown complete")
}
