package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/config"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/postgres"
)

func main(){ cfg,err:=config.Load(); if err!=nil{slog.Error("invalid configuration","error",err);os.Exit(1)}; ctx,cancel:=context.WithTimeout(context.Background(),2*time.Minute); defer cancel(); pool,err:=postgres.Open(ctx,cfg.DatabaseURL); if err!=nil{slog.Error("database unavailable","error",err);os.Exit(1)}; defer pool.Close(); if err:=migrate.Apply(ctx,pool,cfg.MigrationDir); err!=nil{slog.Error("migration failed","error",err);os.Exit(1)}; slog.Info("migrations applied") }
