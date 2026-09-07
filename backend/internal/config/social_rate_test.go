package config

import (
	"strings"
	"testing"
)

func TestLoadRejectsSocialRateIdleTTLBelowWindow(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgresql://user:password@db.example.test:5432/linkup")
	t.Setenv("LINKUP_SOCIAL_RATE_LIMIT", "120")
	t.Setenv("LINKUP_SOCIAL_RATE_WINDOW", "2m")
	t.Setenv("LINKUP_SOCIAL_RATE_IDLE_TTL", "1m")
	t.Setenv("LINKUP_SOCIAL_RATE_MAX_ENTRIES", "20000")
	t.Setenv("LINKUP_RECOVERY_SMTP_ADDR", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_HOST", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_USERNAME", "")
	t.Setenv("LINKUP_RECOVERY_SMTP_PASSWORD", "")
	t.Setenv("LINKUP_RECOVERY_FROM", "")
	t.Setenv("LINKUP_RECOVERY_RESET_URL", "")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), "LINKUP_SOCIAL_RATE_IDLE_TTL") {
		t.Fatalf("expected social rate idle/window validation error, got %v", err)
	}
}
