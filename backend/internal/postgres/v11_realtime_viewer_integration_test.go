package postgres

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func TestV11RealtimeViewerFeedIntegration(t *testing.T) {
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
	assertMigrationCount(t, ctx, pool, 15)

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
	blocks, err := blocklist.NewService(NewBlockStore(pool, time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	viewerStore, err := NewRealtimeViewerStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	feed, err := realtime.NewFeedService(viewerStore)
	if err != nil {
		t.Fatal(err)
	}

	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accounts, "rvh", suffix)
	member := registerIntegrationUser(t, ctx, accounts, "rvm", suffix)
	stranger := registerIntegrationUser(t, ctx, accounts, "rvs", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, member.User.ID, stranger.User.ID}) })

	before := realtimeMaxSequence(t, ctx, pool)
	created, err := slots.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Viewer Feed", Activity: "coffee", PlaceText: "Viewer test place", Capacity: 3,
	}, "viewer-feed-create-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slots.Request(ctx, member.User.ID, created.ID, "viewer-feed-request-"+suffix); err != nil {
		t.Fatal(err)
	}
	approved, err := slots.Approve(ctx, host.User.ID, created.ID, member.User.ID, "viewer-feed-approve-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := chats.Send(ctx, member.User.ID, created.ID, "viewer-feed-chat-"+suffix, "private coordination"); err != nil {
		t.Fatal(err)
	}

	strangerBatch, err := feed.Pull(ctx, stranger.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRealtimeType(strangerBatch.Events, "slot.created") {
		t.Fatalf("public lifecycle invalidation missing for stranger: %#v", strangerBatch.Events)
	}
	for _, privateType := range []string{"slot.request_created", "slot.request_removed", "slot.membership_added", "slot.chat_message_created"} {
		if hasRealtimeType(strangerBatch.Events, privateType) {
			t.Fatalf("stranger received private %s event: %#v", privateType, strangerBatch.Events)
		}
	}

	hostBatch, err := feed.Pull(ctx, host.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"slot.request_created", "slot.membership_added", "slot.chat_message_created"} {
		if !hasRealtimeType(hostBatch.Events, want) {
			t.Fatalf("host missing %s: %#v", want, hostBatch.Events)
		}
	}
	memberBatch, err := feed.Pull(ctx, member.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRealtimeType(memberBatch.Events, "slot.chat_message_created") {
		t.Fatalf("accepted member missing chat invalidation: %#v", memberBatch.Events)
	}

	blockStart := realtimeMaxSequence(t, ctx, pool)
	if err := blocks.Block(ctx, host.User.ID, stranger.User.ID); err != nil {
		t.Fatal(err)
	}
	blockedBatch, err := feed.Pull(ctx, stranger.User.ID, blockStart, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRealtimeType(blockedBatch.Events, "user.block_created") {
		t.Fatalf("blocked user must receive revocation invalidation: %#v", blockedBatch.Events)
	}

	title := "Viewer Feed Updated"
	if _, err := slots.Edit(ctx, host.User.ID, created.ID, slot.EditInput{
		ExpectedVersion: approved.Version,
		Title:           &title,
	}, "viewer-feed-edit-"+suffix); err != nil {
		t.Fatal(err)
	}
	postBlock, err := feed.Pull(ctx, stranger.User.ID, blockedBatch.Cursor, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if hasRealtimeSlotType(postBlock.Events, created.ID, "slot.updated") {
		t.Fatalf("blocked stranger received slot update: %#v", postBlock.Events)
	}
	if postBlock.Cursor <= blockedBatch.Cursor {
		t.Fatalf("cursor did not advance across hidden event: before=%d after=%d", blockedBatch.Cursor, postBlock.Cursor)
	}
}

func realtimeMaxSequence(t *testing.T, ctx context.Context, pool *pgxpool.Pool) int64 {
	t.Helper()
	var sequence int64
	if err := pool.QueryRow(ctx, `SELECT COALESCE(max(sequence),0) FROM domain_outbox_events`).Scan(&sequence); err != nil {
		t.Fatal(err)
	}
	return sequence
}

func hasRealtimeType(events []realtime.Event, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType {
			return true
		}
	}
	return false
}

func hasRealtimeSlotType(events []realtime.Event, slotID, eventType string) bool {
	for _, event := range events {
		if event.Type == eventType && event.SlotID != nil && *event.SlotID == slotID {
			return true
		}
	}
	return false
}
