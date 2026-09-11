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
CREATE TEMP TABLE friendships(user_lo_id text, user_hi_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_selected_viewers(slot_id text, user_id text) ON COMMIT DROP;
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

// TestIdempotencyReplayAuthorizationSQLLinksVisibility closes backend
// IMPLEMENTATION_STATUS.md §56's own explicitly stated gap: the
// "slot.leave" case's PUBLIC-visibility fallback branch (a stranger with
// no current relationship can still replay-read a still-generally-visible
// Slot) needed a LINKS sibling branch gated on a real friendships row,
// exactly like Get/ListPulse/Map/the realtime viewer already have.
func TestIdempotencyReplayAuthorizationSQLLinksVisibility(t *testing.T) {
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
CREATE TEMP TABLE friendships(user_lo_id text, user_hi_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_selected_viewers(slot_id text, user_id text) ON COMMIT DROP;
INSERT INTO pg_temp.slots(id,host_id,state,visibility) VALUES ('slot','host','FILLING','LINKS');`)
	if err != nil {
		t.Fatal(err)
	}

	mustAllow := func(actor string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", "slot.leave"); err != nil {
			t.Fatalf("%s/slot.leave should be allowed: %v", actor, err)
		}
	}
	mustDeny := func(actor string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", "slot.leave"); !errors.Is(err, slot.ErrForbidden) {
			t.Fatalf("%s/slot.leave should be forbidden: %v", actor, err)
		}
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	// A stranger with no relationship and no friendship must not fall back
	// to LINKS visibility (unlike PUBLIC, LINKS requires a real friendship
	// row, not just the visibility value itself).
	mustDeny("stranger")

	// Stored canonically (user_lo_id < user_hi_id lexicographically, same as
	// friend_store.go's real INSERTs) — 'friend' < 'host'.
	exec("INSERT INTO pg_temp.friendships VALUES ('friend','host')")
	mustAllow("friend")

	// A block between the two still wins over the friendship.
	exec("INSERT INTO pg_temp.user_blocks VALUES ('friend','host')")
	mustDeny("friend")
	exec("DELETE FROM pg_temp.user_blocks")
	mustAllow("friend")

	// Leaving the PUBLISHED/FILLING/FULL state range removes the fallback
	// entirely, exactly like the PUBLIC branch already behaves.
	exec("UPDATE pg_temp.slots SET state='ACTIVE' WHERE id='slot'")
	mustDeny("friend")
}

// TestIdempotencyReplayAuthorizationSQLSelectedVisibility mirrors
// TestIdempotencyReplayAuthorizationSQLLinksVisibility for the SELECTED
// allow-list: the "slot.leave" case's fallback branch needed a SELECTED
// sibling gated on a slot_selected_viewers row, exactly like
// Get/ListPulse/Map/the realtime viewer/the city realtime channel already
// have.
func TestIdempotencyReplayAuthorizationSQLSelectedVisibility(t *testing.T) {
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
CREATE TEMP TABLE friendships(user_lo_id text, user_hi_id text) ON COMMIT DROP;
CREATE TEMP TABLE slot_selected_viewers(slot_id text, user_id text) ON COMMIT DROP;
INSERT INTO pg_temp.slots(id,host_id,state,visibility) VALUES ('slot','host','FILLING','SELECTED');`)
	if err != nil {
		t.Fatal(err)
	}

	mustAllow := func(actor string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", "slot.leave"); err != nil {
			t.Fatalf("%s/slot.leave should be allowed: %v", actor, err)
		}
	}
	mustDeny := func(actor string) {
		t.Helper()
		if err := authorizeIdempotencyReplayTx(ctx, tx, actor, "slot", "slot.leave"); !errors.Is(err, slot.ErrForbidden) {
			t.Fatalf("%s/slot.leave should be forbidden: %v", actor, err)
		}
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}

	// A stranger not on the allow-list must not fall back to SELECTED
	// visibility (unlike PUBLIC, SELECTED requires a real allow-list row).
	mustDeny("stranger")

	exec("INSERT INTO pg_temp.slot_selected_viewers VALUES ('slot','selected')")
	mustAllow("selected")

	// A block between the two still wins over being selected.
	exec("INSERT INTO pg_temp.user_blocks VALUES ('selected','host')")
	mustDeny("selected")
	exec("DELETE FROM pg_temp.user_blocks")
	mustAllow("selected")

	// Leaving the PUBLISHED/FILLING/FULL state range removes the fallback
	// entirely, exactly like the PUBLIC/LINKS branches already behave.
	exec("UPDATE pg_temp.slots SET state='ACTIVE' WHERE id='slot'")
	mustDeny("selected")
}
