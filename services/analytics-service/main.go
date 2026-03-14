package main

import (
	"context"
	"encoding/json"
	"log"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

func main() {
	log.Println("[Analytics] Service starting...")

	cfg := loadConfig()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Redis
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("[Analytics] Redis connect failed: %v", err)
	}
	defer rdb.Close()
	log.Println("[Analytics] Connected to Redis")

	tracker := NewWindowTracker(rdb)

	// Kafka consumer
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.KafkaBrokers,
		Topic:   types.TopicPriceUpdates,
		GroupID: "analytics-service",
	})
	defer reader.Close()

	log.Println("[Analytics] Consuming from price-updates (group: analytics-service)")

	for {
		msg, err := reader.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			log.Printf("[Analytics] Read error: %v", err)
			continue
		}

		var event types.PriceUpdateEvent
		if err := json.Unmarshal(msg.Value, &event); err != nil {
			log.Printf("[Analytics] Unmarshal error: %v", err)
			continue
		}

		tracker.Record(ctx, event)
	}

	log.Println("[Analytics] Shutdown complete")
}
