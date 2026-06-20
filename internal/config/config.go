package config

import (
	"log/slog"
	"os"
	"strings"
)

type Config struct {
	Addr     string
	AppEnv   string
	LogLevel slog.Level
	DBURL    string
	Bus      string
	NATSURL  string
}

func Load() Config {
	return Config{
		Addr:     env("ADDR", ":8080"),
		AppEnv:   env("APP_ENV", "dev"),
		LogLevel: parseLevel(env("LOG_LEVEL", "info")),
		DBURL:    env("DB_URL", "postgres://app:app@localhost:"+env("POSTGRES_PORT", "5432")+"/app?sslmode=disable"),
		Bus:      strings.ToLower(env("BUS", "memory")),
		NATSURL:  env("NATS_URL", "nats://localhost:"+env("NATS_PORT", "4222")),
	}
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
