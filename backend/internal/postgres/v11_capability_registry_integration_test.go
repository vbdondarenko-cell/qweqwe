package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
)

func TestV11CapabilityRegistryPostgresIntegration(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires explicit destructive opt-in")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	store, err := NewCapabilityStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	service, err := capability.NewService(store)
	if err != nil {
		t.Fatal(err)
	}

	allowed := "11111111-1111-1111-1111-111111111111"
	other := "22222222-2222-2222-2222-222222222222"
	initial, err := service.Snapshot(ctx, allowed)
	if err != nil {
		t.Fatal(err)
	}
	for key, enabled := range initial.Capabilities {
		if enabled {
			t.Fatalf("default capability %s must be disabled", key)
		}
	}

	if _, err := pool.Exec(ctx, `UPDATE capability_registry
        SET enabled=true, scope_type='USER_ALLOWLIST', scope_user_ids=ARRAY[$1::uuid], reason='integration test'
        WHERE capability_key='realtime'`, allowed); err != nil {
		t.Fatal(err)
	}
	scoped, err := service.Snapshot(ctx, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if !scoped.Capabilities[string(capability.Realtime)] {
		t.Fatal("allowlisted realtime capability not enabled")
	}
	if scoped.Revision <= initial.Revision {
		t.Fatalf("revision did not advance: %d <= %d", scoped.Revision, initial.Revision)
	}

	denied, err := service.Snapshot(ctx, other)
	if err != nil {
		t.Fatal(err)
	}
	if denied.Capabilities[string(capability.Realtime)] {
		t.Fatal("non-allowlisted user received realtime capability")
	}

	beforeDisable := scoped.Revision
	if _, err := pool.Exec(ctx, `UPDATE capability_registry
        SET enabled=false, scope_type='ALL', scope_user_ids='{}'::uuid[], reason='integration reset'
        WHERE capability_key='realtime'`); err != nil {
		t.Fatal(err)
	}
	disabled, err := service.Snapshot(ctx, allowed)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Capabilities[string(capability.Realtime)] {
		t.Fatal("disabled realtime remained enabled")
	}
	if disabled.Revision <= beforeDisable {
		t.Fatalf("disable revision did not advance: %d <= %d", disabled.Revision, beforeDisable)
	}
}
