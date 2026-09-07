package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
    "time"
    "github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"

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
CREATE TEMP TABLE slots(id text PRIMARY KEY, host_id text, state text, accepted_count integer DEFAULT 1, version bigint DEFAULT 1, updated_at timestamptz) ON COMMIT DROP;
CREATE TEMP TABLE app_users(id text PRIMARY KEY, username text, display_name text, avatar_url text) ON COMMIT DROP;
CREATE TEMP TABLE slot_memberships(slot_id text, user_id text) ON COMMIT DROP;
CREATE TEMP TABLE user_blocks(blocker_id text, blocked_id text) ON COMMIT DROP;
INSERT INTO pg_temp.slots(id,host_id,state) VALUES ('link','host','ACTIVE'),('empty','host','PUBLISHED');
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
    exec("UPDATE pg_temp.slots SET state='FULL' WHERE id='link'")
    if err := authorizeChatTx(ctx, tx, "member", "link"); err != nil { t.Fatal(err) }
    if err := removeMemberTx(ctx, tx, "stranger", "link", "member", 1, false, time.Now()); !errors.Is(err,slot.ErrForbidden) { t.Fatal("non-host removal allowed") }
    if err := removeMemberTx(ctx, tx, "host", "link", "member", 2, false, time.Now()); !errors.Is(err,slot.ErrConflict) { t.Fatal("stale removal allowed") }
    if err := removeMemberTx(ctx, tx, "host", "link", "member", 1, false, time.Now()); err != nil { t.Fatal(err) }
    // A replay must not decrement twice and must still authorize the current host.
    if err := removeMemberTx(ctx, tx, "host", "link", "member", 1, true, time.Now()); err != nil { t.Fatal(err) }
    if err := removeMemberTx(ctx, tx, "stranger", "link", "member", 1, true, time.Now()); !errors.Is(err,slot.ErrForbidden) { t.Fatal("unauthorized replay allowed") }
    var count int
    var version int64
    var state string
    if err := tx.QueryRow(ctx,"SELECT accepted_count,version,state FROM pg_temp.slots WHERE id='link'").Scan(&count,&version,&state); err != nil { t.Fatal(err) }
    if count != 0 || version != 2 || state != "FILLING" { t.Fatalf("count=%d version=%d state=%s",count,version,state) }
    if err := authorizeChatTx(ctx, tx, "member", "link"); !errors.Is(err,chat.ErrForbidden) { t.Fatalf("removed member chat access: %v",err) }

	read("host", "link", 0)
}
