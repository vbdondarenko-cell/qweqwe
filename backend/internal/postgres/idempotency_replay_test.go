package postgres

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestIdempotencyReplayAuthorizationSQL(t *testing.T) {
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
CREATE TEMP TABLE slots(id text PRIMARY KEY, host_id text, state text, visibility text) ON COMMIT DROP;
CREATE TEMP TABLE slot_requests(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_memberships(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE user_blocks(blocker_id text, blocked_id text) ON COMMIT DROP;
INSERT INTO pg_temp.slots(id,host_id,state,visibility) VALUES ('slot','host','FILLING','PUBLIC');`)
	if err != nil {
		t.Fatal(err)
	}

	mustAllow := func(actor, operation string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", operation); err != nil {
			t.Fatalf("%s/%s should be allowed: %v", actor, operation, err)
		}
	}
	mustDeny := func(actor, operation string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", operation); !errors.Is(err, slot.ErrForbidden) {
			t.Fatalf("%s/%s should be forbidden: %v", actor, operation, err)
		}
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	// Host mutations can only replay for the current host.
	for _, op := range []string{"slot.create", "slot.edit", "slot.cancel", "slot.approve", "slot.reject", "slot.start", "slot.complete", "slot.remove_member"} {
		mustAllow("host", op)
		mustDeny("stranger", op)
	}

	// REQUEST replay requires a current pending/accepted relationship.
	mustDeny("member", "slot.request")
	exec("INSERT INTO pg_temp.slot_requests VALUES ('slot','member')")
	mustAllow("member", "slot.request")
	exec("DELETE FROM pg_temp.slot_requests WHERE user_id='member'")
	mustDeny("member", "slot.request")
	exec("INSERT INTO pg_temp.slot_memberships VALUES ('slot','member')")
	mustAllow("member", "slot.request")
	exec("INSERT INTO pg_temp.user_blocks VALUES ('host','member')")
	mustDeny("member", "slot.request")
	exec("DELETE FROM pg_temp.user_blocks")

	// LEAVE intentionally removes the relationship, so a replay is allowed while
	// the public Slot remains normally readable, but not after block or ACTIVE
	// access loss. A current accepted membership still authorizes ACTIVE replay.
	exec("DELETE FROM pg_temp.slot_memberships WHERE user_id='member'")
	mustAllow("member", "slot.leave")
	exec("INSERT INTO pg_temp.user_blocks VALUES ('member','host')")
	mustDeny("member", "slot.leave")
	exec("DELETE FROM pg_temp.user_blocks")
	exec("UPDATE pg_temp.slots SET state='ACTIVE' WHERE id='slot'")
	mustDeny("member", "slot.leave")
	exec("INSERT INTO pg_temp.slot_memberships VALUES ('slot','member')")
	mustAllow("member", "slot.leave")

	mustDeny("host", "unknown.operation")
}
