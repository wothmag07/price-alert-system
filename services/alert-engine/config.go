package main

import (
	"github.com/wothmag07/price-alert-system/services/internal/config"
)

type Config struct {
	KafkaBrokers []string
	RedisAddr    string
	PostgresURL  string
}

func loadConfig() Config {
	return Config{
		KafkaBrokers: config.KafkaBrokers(),
		RedisAddr:    config.RedisAddr(),
		PostgresURL:  config.PostgresURL(),
	}
}
