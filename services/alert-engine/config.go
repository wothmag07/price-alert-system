package main

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers []string
	RedisAddr    string
	PostgresURL  string
}

func loadConfig() Config {
	return Config{
		KafkaBrokers: splitComma(envOrDefault("KAFKA_BROKERS", "localhost:9092")),
		RedisAddr:    envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
		PostgresURL:  buildPostgresURL(),
	}
}

func buildPostgresURL() string {
	host := envOrDefault("POSTGRES_HOST", "localhost")
	port := envOrDefault("POSTGRES_PORT", "5432")
	user := envOrDefault("POSTGRES_USER", "postgres")
	pass := envOrDefault("POSTGRES_PASSWORD", "postgres")
	db := envOrDefault("POSTGRES_DB", "price_alerts")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func splitComma(s string) []string {
	parts := strings.Split(s, ",")
	result := make([]string, 0, len(parts))
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
