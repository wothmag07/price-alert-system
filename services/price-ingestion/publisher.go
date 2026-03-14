package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/segmentio/kafka-go"
	"github.com/wothmag07/price-alert-system/services/internal/types"
)

const (
	redisKeyFmt  = "price:latest:%s"
	redisTTL     = 60 * time.Second
	dbThrottleMs = 10_000
)

// Publisher fans out price events to Kafka, Redis, and PostgreSQL.
type Publisher struct {
	kafkaWriter *kafka.Writer
	redisClient *redis.Client
	pgPool      *pgxpool.Pool

	mu          sync.Mutex
	lastDBWrite map[string]int64 // symbol -> last write timestamp (unix ms)
}

// NewPublisher creates connections to Kafka, Redis, and PostgreSQL.
// It pings Redis and PostgreSQL to fail fast if infrastructure is down.
func NewPublisher(ctx context.Context, cfg Config) (*Publisher, error) {
	// Kafka writer
	w := &kafka.Writer{
		Addr:                   kafka.TCP(cfg.KafkaBrokers...),
		Topic:                  types.TopicPriceUpdates,
		Balancer:               &kafka.LeastBytes{},
		AllowAutoTopicCreation: true,
	}

	// Redis client
	rdb := redis.NewClient(&redis.Options{
		Addr: cfg.RedisAddr,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	// PostgreSQL pool
	pool, err := pgxpool.New(ctx, cfg.PostgresURL)
	if err != nil {
		return nil, fmt.Errorf("postgres connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("postgres ping: %w", err)
	}

	return &Publisher{
		kafkaWriter: w,
		redisClient: rdb,
		pgPool:      pool,
		lastDBWrite: make(map[string]int64),
	}, nil
}

// Publish sends a price event to Kafka, Redis, and PostgreSQL in parallel.
func (p *Publisher) Publish(ctx context.Context, event *types.PriceUpdateEvent) {
	jsonBytes, err := json.Marshal(event)
	if err != nil {
		log.Printf("[Publisher] JSON marshal error: %v", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// Kafka
	go func() {
		defer wg.Done()
		err := p.kafkaWriter.WriteMessages(ctx, kafka.Message{
			Key:   []byte(event.Symbol),
			Value: jsonBytes,
		})
		if err != nil {
			log.Printf("[Publisher] Kafka error: %v", err)
		}
	}()

	// Redis
	go func() {
		defer wg.Done()
		key := fmt.Sprintf(redisKeyFmt, event.Symbol)
		err := p.redisClient.Set(ctx, key, jsonBytes, redisTTL).Err()
		if err != nil {
			log.Printf("[Publisher] Redis error: %v", err)
		}
	}()

	// PostgreSQL (throttled)
	go func() {
		defer wg.Done()
		p.maybePersist(ctx, event)
	}()

	wg.Wait()
}

// maybePersist writes to price_history at most once per 10 seconds per symbol.
func (p *Publisher) maybePersist(ctx context.Context, event *types.PriceUpdateEvent) {
	p.mu.Lock()
	lastWrite := p.lastDBWrite[event.Symbol]
	if event.Timestamp-lastWrite < dbThrottleMs {
		p.mu.Unlock()
		return
	}
	p.lastDBWrite[event.Symbol] = event.Timestamp
	p.mu.Unlock()

	_, err := p.pgPool.Exec(ctx,
		`INSERT INTO price_history (symbol, price, volume, timestamp) VALUES ($1, $2, $3, $4)`,
		event.Symbol, event.Price, event.Volume, time.UnixMilli(event.Timestamp),
	)
	if err != nil {
		log.Printf("[Publisher] PostgreSQL error: %v", err)
	}
}

// Close shuts down all publisher connections.
func (p *Publisher) Close() {
	if err := p.kafkaWriter.Close(); err != nil {
		log.Printf("[Publisher] Kafka close error: %v", err)
	}
	if err := p.redisClient.Close(); err != nil {
		log.Printf("[Publisher] Redis close error: %v", err)
	}
	p.pgPool.Close()
}
