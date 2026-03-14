package main

import (
	"github.com/wothmag07/price-alert-system/services/internal/config"
)

type Config struct {
	Port           string
	PostgresURL    string
	RedisAddr      string
	KafkaBrokers   string
	JWTSecret      string
	JWTExpiresMin  int
	JWTRefreshDays int
}

func LoadConfig() Config {
	return Config{
		Port:           config.EnvOrDefault("API_PORT", "3000"),
		PostgresURL:    config.PostgresURL(),
		RedisAddr:      config.RedisAddr(),
		KafkaBrokers:   config.EnvOrDefault("KAFKA_BROKERS", "localhost:9092"),
		JWTSecret:      config.EnvOrDefault("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiresMin:  15,
		JWTRefreshDays: 7,
	}
}
