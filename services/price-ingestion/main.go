package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/wothmag07/price-alert-system/services/internal/types"
)

func main() {
	log.Println("[Price Ingestion] Service starting...")

	cfg := loadConfig()

	// Create a root context that cancels on SIGTERM/SIGINT
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	// Initialize publishers (fail-fast if infra is down)
	pub, err := NewPublisher(ctx, cfg)
	if err != nil {
		log.Fatalf("[Price Ingestion] Failed to connect publishers: %v", err)
	}
	defer pub.Close()
	log.Println("[Price Ingestion] Connected to Kafka, Redis, PostgreSQL")

	log.Printf("[Price Ingestion] Tracking symbols: %v", cfg.TrackedSymbols)
	log.Printf("[Price Ingestion] Publishing to topic: %s", types.TopicPriceUpdates)

	// Create a buffered channel for WebSocket messages
	messages := make(chan []byte, 256)

	// Start the Binance WebSocket client in a goroutine
	go connectBinance(ctx, cfg, messages)

	// Process messages until context is cancelled
	for {
		select {
		case <-ctx.Done():
			log.Println("[Price Ingestion] Shutting down...")
			log.Println("[Price Ingestion] Shutdown complete")
			return
		case raw := <-messages:
			event := parseMiniTicker(raw)
			if event != nil {
				go pub.Publish(ctx, event)
			}
		}
	}
}
