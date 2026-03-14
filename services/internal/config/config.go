package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// EnvOrDefault returns the value of the environment variable or the fallback.
func EnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// EnvIntOrDefault returns the integer value of the environment variable or the fallback.
func EnvIntOrDefault(key string, fallback int) int {
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

// SplitComma splits a comma-separated string into a trimmed slice.
func SplitComma(s string) []string {
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

// RedisAddr builds a Redis address from environment variables.
func RedisAddr() string {
	return EnvOrDefault("REDIS_HOST", "localhost") + ":" + EnvOrDefault("REDIS_PORT", "6379")
}

// KafkaBrokers returns the Kafka broker list from environment variables.
func KafkaBrokers() []string {
	return SplitComma(EnvOrDefault("KAFKA_BROKERS", "localhost:9092"))
}

// PostgresURL builds a PostgreSQL connection string from environment variables.
func PostgresURL() string {
	host := EnvOrDefault("POSTGRES_HOST", "localhost")
	port := EnvOrDefault("POSTGRES_PORT", "5432")
	user := EnvOrDefault("POSTGRES_USER", "postgres")
	pass := EnvOrDefault("POSTGRES_PASSWORD", "postgres")
	db := EnvOrDefault("POSTGRES_DB", "price_alerts")
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, db)
}
