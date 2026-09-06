package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	HTTPAddr        string
	DatabaseURL     string
	SessionTTL      time.Duration
	MigrationDir    string
	ArgonMemoryKiB  uint32
	ArgonIterations uint32
	ArgonParallel   uint8
}

func Load() (Config, error) {
	cfg := Config{
		HTTPAddr:        envOr("LINKUP_HTTP_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		SessionTTL:      30 * 24 * time.Hour,
		MigrationDir:    envOr("LINKUP_MIGRATIONS_DIR", "../db/migrations"),
		ArgonMemoryKiB:  19 * 1024,
		ArgonIterations: 2,
		ArgonParallel:   1,
	}
	if cfg.DatabaseURL == "" {
		return Config{}, errors.New("DATABASE_URL is required")
	}
	var err error
	if cfg.SessionTTL, err = durationEnv("LINKUP_SESSION_TTL", cfg.SessionTTL); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func durationEnv(key string, fallback time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	if seconds, err := strconv.ParseInt(v, 10, 64); err == nil && seconds > 0 {
		return time.Duration(seconds) * time.Second, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return 0, errors.New(key + " must be a positive duration or integer seconds")
	}
	return d, nil
}
