package main

import (
	"github.com/wothmag07/price-alert-system/services/internal/config"
)

type Config struct {
	KafkaBrokers     []string
	RedisAddr        string
	PostgresURL      string
	ResendAPIKey     string
	FromEmail        string
	RateLimitPerHour int
}

func loadConfig() Config {
	return Config{
		KafkaBrokers:     config.KafkaBrokers(),
		RedisAddr:        config.RedisAddr(),
		PostgresURL:      config.PostgresURL(),
		ResendAPIKey:     config.EnvOrDefault("RESEND_API_KEY", ""),
		FromEmail:        config.EnvOrDefault("FROM_EMAIL", "alerts@pricealert.dev"),
		RateLimitPerHour: 10,
	}
}
