package main

import "testing"

func TestMigrationConfigRequiresDedicatedDSN(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://runtime.example.invalid/linkup")
	t.Setenv("LINKUP_MIGRATION_DATABASE_URL", "")
	if _, err := loadMigrationRuntimeConfig(); err == nil {
		t.Fatal("runtime DATABASE_URL must never authorize schema migration")
	}
}

func TestMigrationConfigUsesDedicatedDSNAndDefaultDir(t *testing.T) {
	t.Setenv("LINKUP_MIGRATION_DATABASE_URL", "postgresql://migrator.example.invalid/linkup")
	t.Setenv("LINKUP_MIGRATIONS_DIR", "")
	cfg, err := loadMigrationRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DatabaseURL != "postgresql://migrator.example.invalid/linkup" || cfg.MigrationDir != "../db/migrations" {
		t.Fatalf("unexpected migration config: %#v", cfg)
	}
}

func TestMigrationConfigHonorsExplicitDir(t *testing.T) {
	t.Setenv("LINKUP_MIGRATION_DATABASE_URL", "postgresql://migrator.example.invalid/linkup")
	t.Setenv("LINKUP_MIGRATIONS_DIR", "/srv/linkup/migrations")
	cfg, err := loadMigrationRuntimeConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MigrationDir != "/srv/linkup/migrations" {
		t.Fatalf("migration dir=%q", cfg.MigrationDir)
	}
}
