package main

import (
	"os"
	"strings"
)

type Config struct {
	KafkaBrokers []string
	RedisAddr    string
}

func loadConfig() Config {
	return Config{
		KafkaBrokers: splitComma(envOrDefault("KAFKA_BROKERS", "localhost:9092")),
		RedisAddr:    envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
	}
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
