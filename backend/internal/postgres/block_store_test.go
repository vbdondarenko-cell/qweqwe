package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
)

func TestBlockPairTxRevokesRelationshipsAndAdvancesAffectedVersions(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LINKUP_TEST_DATABASE_URL is required for PostgreSQL query tests")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
CREATE TEMP TABLE app_users(id text PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE user_blocks(blocker_id text, blocked_id text, created_at timestamptz, PRIMARY KEY(blocker_id,blocked_id)) ON COMMIT DROP;
CREATE TEMP TABLE slots(id text PRIMARY KEY, host_id text, state text, access_mode text, accepted_count integer, version bigint, updated_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE slot_requests(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_memberships(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_messages(id uuid PRIMARY KEY, slot_id text, kind text, author_id text, body text, idempotency_key text, system_event_type text, subject_user_id text, created_at timestamptz DEFAULT now()) ON COMMIT DROP;
INSERT INTO pg_temp.app_users(id) VALUES ('host'),('member'),('other');
INSERT INTO pg_temp.slots(id,host_id,state,access_mode,accepted_count,version) VALUES
  ('pending-slot','host','FILLING','APPROVAL',0,1),
  ('accepted-slot','host','FULL','APPROVAL',1,7),
  ('unrelated-slot','other','FILLING','APPROVAL',0,3);
INSERT INTO pg_temp.slot_requests(slot_id,user_id) VALUES ('pending-slot','member');
INSERT INTO pg_temp.slot_memberships(slot_id,user_id) VALUES ('accepted-slot','member');`)
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_800_000_000, 0).UTC()
	if err := blockPairTx(ctx, tx, "host", "member", now, time.Hour); err != nil {
		t.Fatal(err)
	}

	var blocks, requests, memberships int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_temp.user_blocks WHERE blocker_id='host' AND blocked_id='member'`).Scan(&blocks); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_temp.slot_requests`).Scan(&requests); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_temp.slot_memberships`).Scan(&memberships); err != nil {
		t.Fatal(err)
	}
	if blocks != 1 || requests != 0 || memberships != 0 {
		t.Fatalf("blocks=%d requests=%d memberships=%d", blocks, requests, memberships)
	}

	// The removed accepted membership gets exactly one MEMBER_LEFT system
	// notice; the pending-only request removal does not (nobody was a
	// member of pending-slot to announce as having left).
	var systemEventType, subjectUserID string
	if err := tx.QueryRow(ctx, `SELECT system_event_type,subject_user_id FROM pg_temp.slot_messages WHERE slot_id='accepted-slot'`).Scan(&systemEventType, &subjectUserID); err != nil {
		t.Fatal(err)
	}
	if systemEventType != "MEMBER_LEFT" || subjectUserID != "member" {
		t.Fatalf("unexpected system message: event=%s subject=%s", systemEventType, subjectUserID)
	}
	var pendingSlotMessages int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_temp.slot_messages WHERE slot_id='pending-slot'`).Scan(&pendingSlotMessages); err != nil {
		t.Fatal(err)
	}
	if pendingSlotMessages != 0 {
		t.Fatalf("pending-only request removal must not emit a system message, got %d", pendingSlotMessages)
	}

	assertSlot := func(id, wantState string, wantCount int, wantVersion int64) {
		t.Helper()
		var state string
		var count int
		var version int64
		var updatedAt time.Time
		if err := tx.QueryRow(ctx, `SELECT state,accepted_count,version,updated_at FROM pg_temp.slots WHERE id=$1`, id).Scan(&state, &count, &version, &updatedAt); err != nil {
			t.Fatal(err)
		}
		if state != wantState || count != wantCount || version != wantVersion || !updatedAt.Equal(now) {
			t.Fatalf("%s state=%s count=%d version=%d updated=%s", id, state, count, version, updatedAt)
		}
	}
	assertSlot("pending-slot", "FILLING", 0, 2)
	assertSlot("accepted-slot", "FILLING", 0, 8)

	var unrelatedVersion int64
	if err := tx.QueryRow(ctx, `SELECT version FROM pg_temp.slots WHERE id='unrelated-slot'`).Scan(&unrelatedVersion); err != nil {
		t.Fatal(err)
	}
	if unrelatedVersion != 3 {
		t.Fatalf("unrelated slot version changed: %d", unrelatedVersion)
	}

	// Idempotent repeat without a new relationship must not advance versions again.
	if err := blockPairTx(ctx, tx, "host", "member", now.Add(time.Second), time.Hour); err != nil {
		t.Fatal(err)
	}
	assertSlot("pending-slot", "FILLING", 0, 2)
	assertSlot("accepted-slot", "FILLING", 0, 8)

	// The idempotent repeat found no membership left to remove, so it must
	// not emit a second MEMBER_LEFT for the same departure.
	var totalSystemMessages int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_temp.slot_messages WHERE slot_id='accepted-slot'`).Scan(&totalSystemMessages); err != nil {
		t.Fatal(err)
	}
	if totalSystemMessages != 1 {
		t.Fatalf("idempotent block repeat must not duplicate the MEMBER_LEFT notice, got %d", totalSystemMessages)
	}
}

func TestBlockPairTxRejectsMissingTarget(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LINKUP_TEST_DATABASE_URL is required for PostgreSQL query tests")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close(ctx)
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	_, err = tx.Exec(ctx, `
CREATE TEMP TABLE app_users(id text PRIMARY KEY) ON COMMIT DROP;
CREATE TEMP TABLE user_blocks(blocker_id text, blocked_id text, created_at timestamptz, PRIMARY KEY(blocker_id,blocked_id)) ON COMMIT DROP;
CREATE TEMP TABLE slots(id text PRIMARY KEY, host_id text, state text, access_mode text, accepted_count integer, version bigint, updated_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE slot_requests(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_memberships(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_messages(id uuid PRIMARY KEY, slot_id text, kind text, author_id text, body text, idempotency_key text, system_event_type text, subject_user_id text, created_at timestamptz DEFAULT now()) ON COMMIT DROP;
INSERT INTO pg_temp.app_users(id) VALUES ('host');`)
	if err != nil {
		t.Fatal(err)
	}
	if err := blockPairTx(ctx, tx, "host", "missing", time.Now(), time.Hour); !errors.Is(err, blocklist.ErrInvalidTarget) {
		t.Fatalf("expected invalid target, got %v", err)
	}
}
