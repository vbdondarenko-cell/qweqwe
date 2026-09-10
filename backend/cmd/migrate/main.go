package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/postgres"
)

type migrationRuntimeConfig struct {
	DatabaseURL  string
	MigrationDir string
}

func loadMigrationRuntimeConfig() (migrationRuntimeConfig, error) {
	dsn := strings.TrimSpace(os.Getenv("LINKUP_MIGRATION_DATABASE_URL"))
	if dsn == "" {
		return migrationRuntimeConfig{}, errors.New("LINKUP_MIGRATION_DATABASE_URL is required")
	}
	dir := strings.TrimSpace(os.Getenv("LINKUP_MIGRATIONS_DIR"))
	if dir == "" {
		dir = "../db/migrations"
	}
	return migrationRuntimeConfig{DatabaseURL: dsn, MigrationDir: dir}, nil
}

func main() {
	cfg, err := loadMigrationRuntimeConfig()
	if err != nil {
		slog.Error("invalid migration configuration", "error", err)
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := postgres.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Error("database unavailable", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, cfg.MigrationDir); err != nil {
		slog.Error("migration failed", "error", err)
		os.Exit(1)
	}
	slog.Info("migrations applied")
}
