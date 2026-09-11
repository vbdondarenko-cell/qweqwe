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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11ChatSystemMessageOnSlotStarted covers README §6.7 "system messages
// for important Slot lifecycle events" for the SLOT_STARTED case: starting a
// Slot must record a SYSTEM chat notice, in the same transaction as the
// state transition, visible to host and accepted members through the
// existing chat read path with no author and a Subject naming the host.
func TestV11ChatSystemMessageOnSlotStarted(t *testing.T) {
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
	t.Cleanup(pool.Close)
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "csh", suffix)
	member := registerIntegrationUser(t, ctx, accountService, "csm", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, member.User.ID}) })

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}
	chatService, err := chat.NewService(NewChatStore(pool))
	if err != nil {
		t.Fatal(err)
	}

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "System Message Coffee", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "v11-chat-system-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "v11-chat-system-request-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member.User.ID, "v11-chat-system-approve-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := chatService.Send(ctx, member.User.ID, created.ID, "v11-chat-system-usermsg-0001", "On my way"); err != nil {
		t.Fatal(err)
	}

	active, err := slotService.Start(ctx, host.User.ID, created.ID, "v11-chat-system-start-0001")
	if err != nil {
		t.Fatal(err)
	}
	if active.State != slot.StateActive {
		t.Fatalf("expected ACTIVE, got %#v", active)
	}

	for _, viewer := range []string{host.User.ID, member.User.ID} {
		messages, err := chatService.ListRecent(ctx, viewer, created.ID, 100)
		if err != nil {
			t.Fatalf("viewer %s: %v", viewer, err)
		}
		if len(messages) != 2 {
			t.Fatalf("viewer %s: expected 2 messages (user + system), got %#v", viewer, messages)
		}
		userMsg, systemMsg := messages[0], messages[1]
		if userMsg.Kind != chat.KindUser || userMsg.Author == nil || userMsg.Author.ID != member.User.ID || userMsg.Text != "On my way" {
			t.Fatalf("viewer %s: unexpected user message: %#v", viewer, userMsg)
		}
		if systemMsg.Kind != chat.KindSystem {
			t.Fatalf("viewer %s: expected SYSTEM kind, got %#v", viewer, systemMsg)
		}
		if systemMsg.Author != nil {
			t.Fatalf("viewer %s: system message must have no author, got %#v", viewer, systemMsg.Author)
		}
		if systemMsg.Text != "" {
			t.Fatalf("viewer %s: system message must have no free text body, got %q", viewer, systemMsg.Text)
		}
		if systemMsg.SystemEventType == nil || *systemMsg.SystemEventType != chat.SystemEventSlotStarted {
			t.Fatalf("viewer %s: expected SLOT_STARTED event type, got %#v", viewer, systemMsg.SystemEventType)
		}
		if systemMsg.Subject == nil || systemMsg.Subject.ID != host.User.ID {
			t.Fatalf("viewer %s: expected subject=host, got %#v", viewer, systemMsg.Subject)
		}
		if !systemMsg.CreatedAt.After(userMsg.CreatedAt) && systemMsg.CreatedAt != userMsg.CreatedAt {
			t.Fatalf("viewer %s: system message must not precede the user message it follows: %#v vs %#v", viewer, systemMsg, userMsg)
		}
	}

	// A stranger still has no chat access at all, system messages included.
	stranger := registerIntegrationUser(t, ctx, accountService, "css", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{stranger.User.ID}) })
	if _, err := chatService.ListRecent(ctx, stranger.User.ID, created.ID, 100); !errors.Is(err, chat.ErrForbidden) {
		t.Fatalf("stranger must not see system messages either, got %v", err)
	}

	// Completing the Slot purges every row, system messages included, via the
	// existing unconditional terminal purge trigger.
	if _, err := slotService.Complete(ctx, host.User.ID, created.ID, "v11-chat-system-complete-0001"); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_messages WHERE slot_id=$1`, created.ID).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("expected terminal purge to remove all rows including SYSTEM, got %d", remaining)
	}
}
