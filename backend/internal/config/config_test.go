package config

import (
	"strings"
	"testing"
	"time"
)

func TestDurationEnvRejectsIntegerSecondsOverflow(t *testing.T) {
	t.Setenv("LINKUP_TEST_DURATION", "9223372037")
	if _, err := durationEnv("LINKUP_TEST_DURATION", time.Minute); err == nil {
		t.Fatal("integer seconds that overflow time.Duration must fail")
	}
}

func TestLoadRejectsArgonSafetyCeiling(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:password@db.example.test:5432/linkup")
	t.Setenv("LINKUP_SESSION_TTL", "720h")
	t.Setenv("LINKUP_PASSWORD_RESET_TTL", "30m")
	t.Setenv("LINKUP_IDEMPOTENCY_TTL", "24h")
	t.Setenv("LINKUP_ARGON_MEMORY_KIB", "262145")
	t.Setenv("LINKUP_ARGON_ITERATIONS", "2")
	t.Setenv("LINKUP_ARGON_PARALLELISM", "1")
	t.Setenv("LINKUP_AUTH_RATE_LIMIT", "10")
	t.Setenv("LINKUP_AUTH_RATE_WINDOW", "1m")
	t.Setenv("LINKUP_AUTH_RATE_IDLE_TTL", "10m")
	t.Setenv("LINKUP_AUTH_RATE_MAX_ENTRIES", "20000")
	t.Setenv("LINKUP_RECOVERY_SMTP_ADDR", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_HOST", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_USERNAME", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_PASSWORD", "")
	t.Setenv("LINKUP_RECOVERY_FROM", "")
	t.Setenv("LINKUP_RECOVERY_RESET_URL", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_IMPLICIT_TLS", "false")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "safety ceiling") {
		t.Fatalf("expected Argon2id safety ceiling error, got %v", err)
	}
}
