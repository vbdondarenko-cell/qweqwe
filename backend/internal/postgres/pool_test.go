package postgres

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

// TestBuildPoolConfigUsesSimpleProtocol is a regression guard for backend
// IMPLEMENTATION_STATUS.md §58: production connects through Supabase's
// PgBouncer pooler in transaction-pooling mode, which is incompatible with
// pgx's default server-side prepared-statement caching (observed live as
// repeated "prepared statement ... already exists" errors from
// NotificationProjector's polling ticker). If this ever silently reverts to
// the default QueryExecModeCacheStatement, that class of failure returns.
func TestBuildPoolConfigUsesSimpleProtocol(t *testing.T) {
	cfg, err := buildPoolConfig("postgresql://user:pass@localhost:6543/db")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ConnConfig.DefaultQueryExecMode != pgx.QueryExecModeSimpleProtocol {
		t.Fatalf("expected QueryExecModeSimpleProtocol, got %v", cfg.ConnConfig.DefaultQueryExecMode)
	}
}

func TestOpenRedactsMalformedDatabaseURL(t *testing.T) {
	const secret = "supersecret"
	_, err := Open(context.Background(), "postgresql://user:"+secret+"@%zz")
	if err == nil {
		t.Fatal("malformed database URL must fail")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "%zz") {
		t.Fatalf("database error leaked raw connection material: %q", err)
	}
	if got := err.Error(); got != "invalid database configuration" {
		t.Fatalf("unexpected safe database error: %q", got)
	}
}
