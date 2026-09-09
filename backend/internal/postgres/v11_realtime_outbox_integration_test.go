package postgres

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11RealtimeOutboxIntegration(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatalf("apply canonical migrations: %v", err)
	}
	assertMigrationCount(t, ctx, pool, 14)

	t.Run("runtime api role cannot mutate outbox", func(t *testing.T) {
		var roleExists bool
		if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api')`).Scan(&roleExists); err != nil {
			t.Fatal(err)
		}
		if !roleExists {
			t.Skip("linkup_api role is not present in disposable PostgreSQL")
		}
		var canSelect, canInsert, canUpdate, canDelete, canCursorWrite, canReceiptWrite, canSequence, canEnqueue, canTrigger bool
		err := pool.QueryRow(ctx, `SELECT
			has_table_privilege('linkup_api','public.domain_outbox_events','SELECT'),
			has_table_privilege('linkup_api','public.domain_outbox_events','INSERT'),
			has_table_privilege('linkup_api','public.domain_outbox_events','UPDATE'),
			has_table_privilege('linkup_api','public.domain_outbox_events','DELETE'),
			has_table_privilege('linkup_api','public.connector_cursors','INSERT') OR has_table_privilege('linkup_api','public.connector_cursors','UPDATE') OR has_table_privilege('linkup_api','public.connector_cursors','DELETE'),
			has_table_privilege('linkup_api','public.connector_delivery_receipts','INSERT') OR has_table_privilege('linkup_api','public.connector_delivery_receipts','UPDATE') OR has_table_privilege('linkup_api','public.connector_delivery_receipts','DELETE'),
			has_sequence_privilege('linkup_api','public.domain_outbox_events_sequence_seq','USAGE'),
			has_function_privilege('linkup_api','public.linkup_enqueue_outbox(text,text,uuid,uuid,uuid,jsonb)','EXECUTE'),
			has_function_privilege('linkup_api','public.linkup_emit_canonical_outbox()','EXECUTE')`).
			Scan(&canSelect, &canInsert, &canUpdate, &canDelete, &canCursorWrite, &canReceiptWrite, &canSequence, &canEnqueue, &canTrigger)
		if err != nil {
			t.Fatal(err)
		}
		if !canSelect || canInsert || canUpdate || canDelete || canCursorWrite || canReceiptWrite || canSequence || canEnqueue || canTrigger {
			t.Fatalf("unexpected linkup_api realtime privileges select=%v insert=%v update=%v delete=%v cursorWrite=%v receiptWrite=%v sequence=%v enqueue=%v trigger=%v",
				canSelect, canInsert, canUpdate, canDelete, canCursorWrite, canReceiptWrite, canSequence, canEnqueue, canTrigger)
		}
	})

	accounts, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := slot.NewService(slotStore)
	if err != nil {
		t.Fatal(err)
	}
	chats, err := chat.NewService(NewChatStore(pool))
	if err != nil {
		t.Fatal(err)
	}
	outbox, err := NewRealtimeOutboxStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accounts, "rh", suffix)
	member := registerIntegrationUser(t, ctx, accounts, "rm", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, member.User.ID}) })

	var before int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(max(sequence),0) FROM domain_outbox_events`).Scan(&before); err != nil {
		t.Fatal(err)
	}

	created, err := slots.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Realtime Integration", Activity: "coffee", PlaceText: "Canonical test place", Capacity: 2,
	}, "v11-realtime-create-0001-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slots.Request(ctx, member.User.ID, created.ID, "v11-realtime-request-0001-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := slots.Approve(ctx, host.User.ID, created.ID, member.User.ID, "v11-realtime-approve-0001-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := chats.Send(ctx, member.User.ID, created.ID, "v11-realtime-chat-0000001-"+suffix, "Realtime outbox proof"); err != nil {
		t.Fatal(err)
	}

	events, err := outbox.ListAfter(ctx, before, 100)
	if err != nil {
		t.Fatal(err)
	}
	wantTypes := map[string]bool{
		"slot.created":              false,
		"slot.request_created":      false,
		"slot.request_removed":      false,
		"slot.membership_added":     false,
		"slot.chat_message_created": false,
	}
	for _, event := range events {
		if _, ok := wantTypes[event.Type]; ok {
			wantTypes[event.Type] = true
		}
	}
	for eventType, seen := range wantTypes {
		if !seen {
			t.Fatalf("canonical mutation did not emit %s: %#v", eventType, events)
		}
	}

	all, err := outbox.ListAfter(ctx, 0, realtime.MaxBatch)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 || all[0].Sequence != 1 {
		t.Fatalf("outbox must preserve a catch-up sequence from 1, got %#v", all)
	}
	connector := "v11.test." + suffix
	cursor, err := outbox.Cursor(ctx, connector)
	if err != nil || cursor != 0 {
		t.Fatalf("new connector cursor=%d err=%v", cursor, err)
	}
	if err := outbox.Checkpoint(ctx, connector, all[0], realtime.OutcomeDelivered); err != nil {
		t.Fatalf("checkpoint first event: %v", err)
	}
	if err := outbox.Checkpoint(ctx, connector, all[0], realtime.OutcomeDelivered); err != nil {
		t.Fatalf("idempotent checkpoint replay: %v", err)
	}
	if err := outbox.Checkpoint(ctx, connector, all[0], realtime.OutcomeSkipped); !errors.Is(err, realtime.ErrReceiptConflict) {
		t.Fatalf("changed receipt outcome must conflict, got %v", err)
	}
	if len(all) >= 3 {
		if err := outbox.Checkpoint(ctx, connector, all[2], realtime.OutcomeSkipped); !errors.Is(err, realtime.ErrCursorOutOfOrder) {
			t.Fatalf("cursor gap must be rejected, got %v", err)
		}
	}
}
