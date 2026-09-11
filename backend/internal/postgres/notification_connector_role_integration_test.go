package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

// TestNotificationConnectorGrantsWorkUnderTheAPIRole is the test that would
// have caught a real production-blocking regression this session found by
// accident while verifying EVENT_REMINDER: migration 000014 correctly
// granted linkup_api everything RealtimeOutboxStore needs on
// connector_cursors/connector_delivery_receipts, but migration 000021's
// hardening pass later REVOKEd ALL of it (intending a "future dedicated
// worker role" that was never actually created — cmd/api is still the only
// server binary, and it is the same in-process ticker that calls
// Checkpoint under whatever role DATABASE_URL authenticates as). Every
// other test in this package connects as the Postgres superuser
// (LINKUP_TEST_DATABASE_URL), which bypasses role grants entirely, so this
// was invisible to the full green regression suite the Notifications
// feature (§45-§48) was verified under. Migration 000029 restores the
// grants; this test proves it by actually connecting as linkup_api rather
// than reading the SQL and hoping.
//
// This test is opt-in and skips by default: it requires a linkup_api role
// to already exist with LOGIN privilege and CONNECT granted on the target
// disposable database (a one-time local setup step, since — like linkup_api
// itself — no migration creates the role; production provisions it via
// Supabase). Point LINKUP_TEST_LINKUP_API_DATABASE_URL at the SAME
// database as LINKUP_TEST_DATABASE_URL, authenticated as linkup_api.
func TestNotificationConnectorGrantsWorkUnderTheAPIRole(t *testing.T) {
	superDSN := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if superDSN == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	apiDSN := os.Getenv("LINKUP_TEST_LINKUP_API_DATABASE_URL")
	if apiDSN == "" {
		t.Skip("set LINKUP_TEST_LINKUP_API_DATABASE_URL (a linkup_api-authenticated DSN against the same disposable database) to exercise real production role grants; every other test in this package connects as the Postgres superuser and cannot catch a role-grant regression")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	superPool, err := pgxpool.New(ctx, superDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(superPool.Close)
	if err := migrate.Apply(ctx, superPool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	apiPool, err := pgxpool.New(ctx, apiDSN)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(apiPool.Close)
	if err := apiPool.Ping(ctx); err != nil {
		t.Fatalf("could not connect as linkup_api: %v", err)
	}

	outbox, err := NewRealtimeOutboxStore(apiPool)
	if err != nil {
		t.Fatal(err)
	}
	connector := "role-grant-test-" + mustNewID(t)

	// Cursor(): INSERT ... ON CONFLICT DO NOTHING + SELECT on
	// connector_cursors. This alone reproduces the original failure
	// ("permission denied for table connector_cursors") when run against
	// migrations through 000028 only.
	if _, err := outbox.Cursor(ctx, connector); err != nil {
		t.Fatalf("Cursor() as linkup_api: %v (this is exactly the migration-000021 regression migration 000029 exists to fix)", err)
	}

	// This package's integration tests share one live, never-recreated-
	// per-test database, so real prior outbox volume from every earlier
	// test in this run already exists. Fast-forward the fresh connector
	// past all of it via a direct write (test scaffolding, not part of
	// what this test verifies) so the events this test inserts next are
	// the only ones ListAfter needs to return — Checkpoint correctly
	// refuses to skip past a REAL unprocessed row (that is the guarantee
	// realtime_outbox_store_test.go exercises), so without this the test
	// would hit that unrelated, correct rejection instead of proving
	// anything about role grants.
	var baseline int64
	if err := superPool.QueryRow(ctx, `SELECT COALESCE(max(sequence),0) FROM domain_outbox_events`).Scan(&baseline); err != nil {
		t.Fatal(err)
	}
	if _, err := superPool.Exec(ctx, `UPDATE connector_cursors SET last_sequence=$2 WHERE connector=$1`, connector, baseline); err != nil {
		t.Fatal(err)
	}

	// linkup_enqueue_outbox: EXECUTE grant required for ReminderScanner
	// (the new caller in this block) to emit slot.starting_soon events as
	// linkup_api in production.
	eventID := mustNewID(t)
	if _, err := superPool.Exec(ctx, `
		INSERT INTO domain_outbox_events (event_id,event_type,aggregate_type,aggregate_id,payload)
		VALUES ($1::uuid,'test.role_check','test',gen_random_uuid(),'{}'::jsonb)`, eventID); err != nil {
		t.Fatal(err)
	}
	if _, err := apiPool.Exec(ctx, `
		SELECT linkup_enqueue_outbox('test.role_check_direct','test',gen_random_uuid(),NULL,NULL,'{}'::jsonb)`); err != nil {
		t.Fatalf("linkup_enqueue_outbox() as linkup_api: %v", err)
	}

	// ListAfter(): SELECT on domain_outbox_events (unchanged by 000029, but
	// exercised here for completeness of "does the whole read+write cycle
	// work end to end as linkup_api").
	events, err := outbox.ListAfter(ctx, baseline, 50)
	if err != nil {
		t.Fatalf("ListAfter() as linkup_api: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected exactly the 2 events just inserted, got %d", len(events))
	}

	// Checkpoint(): the actual write path NotificationProjector.ProcessBatch
	// calls after each projected event — INSERT/UPDATE connector_cursors,
	// INSERT connector_delivery_receipts. Checkpointed one at a time, like
	// ProcessBatch's own loop: Checkpoint correctly refuses to skip past a
	// real unprocessed row (that guarantee is realtime_outbox_store_test.go's
	// job to exercise), so jumping straight to the last event here would
	// hit that unrelated, correct rejection instead of proving anything
	// about role grants.
	var last realtime.Event
	for _, event := range events {
		if err := outbox.Checkpoint(ctx, connector, event, realtime.OutcomeDelivered); err != nil {
			t.Fatalf("Checkpoint() as linkup_api: %v (this is exactly the migration-000021 regression migration 000029 exists to fix)", err)
		}
		last = event
	}

	after, err := outbox.Cursor(ctx, connector)
	if err != nil {
		t.Fatal(err)
	}
	if after != last.Sequence {
		t.Fatalf("cursor did not advance to the last checkpointed event: got %d, want %d", after, last.Sequence)
	}
}

func mustNewID(t *testing.T) string {
	t.Helper()
	id, err := identifier.NewUUID()
	if err != nil {
		t.Fatal(err)
	}
	return id
}
