package main

import "os"

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
		Port:           envOrDefault("API_PORT", "3000"),
		PostgresURL:    buildPostgresURL(),
		RedisAddr:      envOrDefault("REDIS_HOST", "localhost") + ":" + envOrDefault("REDIS_PORT", "6379"),
		KafkaBrokers:   envOrDefault("KAFKA_BROKERS", "localhost:9092"),
		JWTSecret:      envOrDefault("JWT_SECRET", "dev-secret-change-in-production"),
		JWTExpiresMin:  15,
		JWTRefreshDays: 7,
	}
}

func buildPostgresURL() string {
	host := envOrDefault("POSTGRES_HOST", "localhost")
	port := envOrDefault("POSTGRES_PORT", "5432")
	user := envOrDefault("POSTGRES_USER", "postgres")
	pass := envOrDefault("POSTGRES_PASSWORD", "postgres")
	db := envOrDefault("POSTGRES_DB", "price_alerts")
	return "postgres://" + user + ":" + pass + "@" + host + ":" + port + "/" + db + "?sslmode=disable"
}

func envOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
