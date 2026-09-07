package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// Exercises the production SQL against PostgreSQL using isolated temporary tables.
// This is query-contract coverage; it does not verify production migrations.
func TestAcceptedRosterSQL(t *testing.T) {
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
CREATE TEMP TABLE slots(id text PRIMARY KEY, host_id text, state text) ON COMMIT DROP;
CREATE TEMP TABLE app_users(id text PRIMARY KEY, username text, display_name text, avatar_url text) ON COMMIT DROP;
CREATE TEMP TABLE slot_memberships(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE user_blocks(blocker_id text, blocked_id text) ON COMMIT DROP;
INSERT INTO pg_temp.slots VALUES ('link','host','ACTIVE'),('empty','host','PUBLISHED');
INSERT INTO pg_temp.app_users VALUES ('member','member','Member',NULL),('host','host','Host',NULL);
INSERT INTO pg_temp.slot_memberships VALUES ('link','member');`)
	if err != nil {
		t.Fatal(err)
	}
	read := func(actor, id string, want int) {
		t.Helper()
		var data []byte
		err := tx.QueryRow(ctx, acceptedRosterSQL, actor, id).Scan(&data)
		if want < 0 {
			if !errors.Is(err, pgx.ErrNoRows) {
				t.Fatalf("%s/%s: expected no authorized row, got %v", actor, id, err)
			}
			return
		}
		if err != nil {
			t.Fatal(err)
		}
		var items []slot.Organizer
		if err := json.Unmarshal(data, &items); err != nil {
			t.Fatal(err)
		}
		if items == nil || len(items) != want {
			t.Fatalf("%s/%s: items=%s want=%d", actor, id, data, want)
		}
		if want == 1 && items[0].ID != "member" {
			t.Fatalf("unexpected identity: %s", data)
		}
	}
	exec := func(query string, args ...any) {
		t.Helper()
		if _, err := tx.Exec(ctx, query, args...); err != nil {
			t.Fatal(err)
		}
	}
	read("host", "link", 1)
	read("host", "empty", 0)
	read("host", "missing", -1)
	for _, actor := range []string{"member", "pending", "stranger"} {
		read(actor, "link", -1)
	}
	for _, pair := range [][2]string{{"host", "member"}, {"member", "host"}} {
		exec("INSERT INTO pg_temp.user_blocks VALUES ($1,$2)", pair[0], pair[1])
		read("host", "link", 0)
		exec("DELETE FROM pg_temp.user_blocks")
	}
	for _, state := range []string{"COMPLETED", "CANCELLED", "EXPIRED", "MODERATED", "DRAFT"} {
		exec("UPDATE pg_temp.slots SET state=$1 WHERE id='link'", state)
		read("host", "link", -1)
	}
	exec("UPDATE pg_temp.slots SET state='FILLING' WHERE id='link'")
	exec("DELETE FROM pg_temp.slot_memberships WHERE user_id='member'")
	read("host", "link", 0)
}
