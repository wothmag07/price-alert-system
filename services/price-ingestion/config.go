package main

import (
	"github.com/wothmag07/price-alert-system/services/internal/config"
)

type Config struct {
	BinanceWsURL   string
	TrackedSymbols []string
	KafkaBrokers   []string
	RedisAddr      string
	PostgresURL    string
}

func loadConfig() Config {
	return Config{
		BinanceWsURL:   config.EnvOrDefault("BINANCE_WS_URL", "wss://stream.binance.com:9443/ws"),
		TrackedSymbols: config.SplitComma(config.EnvOrDefault("TRACKED_SYMBOLS", "btcusdt,ethusdt,solusdt")),
		KafkaBrokers:   config.KafkaBrokers(),
		RedisAddr:      config.RedisAddr(),
		PostgresURL:    config.PostgresURL(),
	}
}
