package main

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	// Binance
	BinanceWsURL   string
	TrackedSymbols []string

	// Kafka
	KafkaBrokers []string

	// Redis
	RedisAddr string

	// PostgreSQL
	PostgresHost     string
	PostgresPort     int
	PostgresDB       string
	PostgresUser     string
	PostgresPassword string
}

func loadConfig() Config {
	return Config{
		BinanceWsURL:   envOrDefault("BINANCE_WS_URL", "wss://stream.binance.com:9443/ws"),
		TrackedSymbols: splitComma(envOrDefault("TRACKED_SYMBOLS", "btcusdt,ethusdt,solusdt")),
		KafkaBrokers:   splitComma(envOrDefault("KAFKA_BROKERS", "localhost:9092")),
		RedisAddr:      envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
		PostgresHost:   envOrDefault("POSTGRES_HOST", "localhost"),
		PostgresPort:   envIntOrDefault("POSTGRES_PORT", 5432),
		PostgresDB:     envOrDefault("POSTGRES_DB", "price_alerts"),
		PostgresUser:   envOrDefault("POSTGRES_USER", "postgres"),
		PostgresPassword: envOrDefault("POSTGRES_PASSWORD", "postgres"),
	}
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envIntOrDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(strings.ToLower(p))
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
