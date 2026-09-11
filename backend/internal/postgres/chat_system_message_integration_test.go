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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11ChatSystemMessageOnSlotStarted covers README §6.7 "system messages
// for important Slot lifecycle events" for the APPROVAL accept (MEMBER_JOINED)
// and SLOT_STARTED cases together, in the order they actually occur: both
// must record a SYSTEM chat notice, in the same transaction as the state
// change they describe, visible to host and accepted members through the
// existing chat read path with no author and a Subject naming who it is about.
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

	assertSystem := func(t *testing.T, viewer string, msg chat.Message, wantEvent chat.SystemEventType, wantSubject string) {
		t.Helper()
		if msg.Kind != chat.KindSystem {
			t.Fatalf("viewer %s: expected SYSTEM kind, got %#v", viewer, msg)
		}
		if msg.Author != nil {
			t.Fatalf("viewer %s: system message must have no author, got %#v", viewer, msg.Author)
		}
		if msg.Text != "" {
			t.Fatalf("viewer %s: system message must have no free text body, got %q", viewer, msg.Text)
		}
		if msg.SystemEventType == nil || *msg.SystemEventType != wantEvent {
			t.Fatalf("viewer %s: expected %s event type, got %#v", viewer, wantEvent, msg.SystemEventType)
		}
		if msg.Subject == nil || msg.Subject.ID != wantSubject {
			t.Fatalf("viewer %s: expected subject=%s, got %#v", viewer, wantSubject, msg.Subject)
		}
	}

	for _, viewer := range []string{host.User.ID, member.User.ID} {
		messages, err := chatService.ListRecent(ctx, viewer, created.ID, 100)
		if err != nil {
			t.Fatalf("viewer %s: %v", viewer, err)
		}
		if len(messages) != 3 {
			t.Fatalf("viewer %s: expected 3 messages (joined + user + started), got %#v", viewer, messages)
		}
		joinedMsg, userMsg, startedMsg := messages[0], messages[1], messages[2]
		assertSystem(t, viewer, joinedMsg, chat.SystemEventMemberJoined, member.User.ID)
		if userMsg.Kind != chat.KindUser || userMsg.Author == nil || userMsg.Author.ID != member.User.ID || userMsg.Text != "On my way" {
			t.Fatalf("viewer %s: unexpected user message: %#v", viewer, userMsg)
		}
		assertSystem(t, viewer, startedMsg, chat.SystemEventSlotStarted, host.User.ID)
		if joinedMsg.CreatedAt.After(userMsg.CreatedAt) || userMsg.CreatedAt.After(startedMsg.CreatedAt) {
			t.Fatalf("viewer %s: messages out of chronological order: %#v", viewer, messages)
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

// TestV11ChatSystemMessageOnMemberLeftAndRemoved covers the two MEMBER_LEFT
// paths that are not the WAITLIST/promotion path exercised elsewhere: a
// member leaving on their own (SlotStore.Leave, shared by v1.0 APPROVAL and
// v1.1 INSTANT) and a host explicitly removing a member (removeMemberTx,
// also shared).
func TestV11ChatSystemMessageOnMemberLeftAndRemoved(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "clh", suffix)
	leaver := registerIntegrationUser(t, ctx, accountService, "cll", suffix)
	removed := registerIntegrationUser(t, ctx, accountService, "clr", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, leaver.User.ID, removed.User.ID}) })

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
		Title: "Leave And Remove", Activity: "coffee", PlaceText: "Center", Capacity: 3,
	}, "v11-chat-system-leave-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []string{leaver.User.ID, removed.User.ID} {
		if _, err := slotService.Request(ctx, member, created.ID, "v11-chat-system-leave-request-"+member); err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member, "v11-chat-system-leave-approve-"+member); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := slotService.Leave(ctx, leaver.User.ID, created.ID, "v11-chat-system-leave-0001"); err != nil {
		t.Fatal(err)
	}
	current, err := slotService.Get(ctx, host.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.RemoveMember(ctx, host.User.ID, created.ID, removed.User.ID, current.Version, "v11-chat-system-remove-0001"); err != nil {
		t.Fatal(err)
	}

	messages, err := chatService.ListRecent(ctx, host.User.ID, created.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	// Two MEMBER_JOINED (approve x2) followed by MEMBER_LEFT (leave) and
	// MEMBER_LEFT (host removal), in that chronological order.
	if len(messages) != 4 {
		t.Fatalf("expected 4 SYSTEM messages, got %#v", messages)
	}
	wantEvents := []struct {
		event   chat.SystemEventType
		subject string
	}{
		{chat.SystemEventMemberJoined, leaver.User.ID},
		{chat.SystemEventMemberJoined, removed.User.ID},
		{chat.SystemEventMemberLeft, leaver.User.ID},
		{chat.SystemEventMemberLeft, removed.User.ID},
	}
	for i, want := range wantEvents {
		msg := messages[i]
		if msg.Kind != chat.KindSystem || msg.SystemEventType == nil || *msg.SystemEventType != want.event || msg.Subject == nil || msg.Subject.ID != want.subject {
			t.Fatalf("message %d: want event=%s subject=%s, got %#v", i, want.event, want.subject, msg)
		}
	}
}

// TestV11ChatSystemMessageOnInstantAndWaitlistJoin covers the two remaining
// MEMBER_JOINED paths: V11SlotStore.Join (INSTANT immediate admission) and
// the WAITLIST promotion path in promoteOldestWaitlistTx (a queued request
// promoted into a seat freed by another member leaving).
func TestV11ChatSystemMessageOnInstantAndWaitlistJoin(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "cih", suffix)
	instantJoiner := registerIntegrationUser(t, ctx, accountService, "cii", suffix)
	waitlistFirst := registerIntegrationUser(t, ctx, accountService, "cwf", suffix)
	waitlistFiller := registerIntegrationUser(t, ctx, accountService, "cwm", suffix)
	waitlistSecond := registerIntegrationUser(t, ctx, accountService, "cws", suffix)
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, instantJoiner.User.ID, waitlistFirst.User.ID, waitlistFiller.User.ID, waitlistSecond.User.ID})
	})

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	v11Store, err := NewV11SlotStore(baseStore, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(v11Store)
	if err != nil {
		t.Fatal(err)
	}
	chatService, err := chat.NewService(NewChatStore(pool))
	if err != nil {
		t.Fatal(err)
	}

	instantMode := slot.AccessInstant
	instantDraft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Instant System Message", Activity: "coffee", PlaceText: "Center", Capacity: 2, AccessMode: &instantMode,
	}, "v11-chat-system-instant-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	instantSlot, err := slotService.PublishDraft(ctx, host.User.ID, instantDraft.ID, instantDraft.Version, "v11-chat-system-instant-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Join(ctx, instantJoiner.User.ID, instantSlot.ID, "v11-chat-system-instant-join-0001"); err != nil {
		t.Fatal(err)
	}
	instantMessages, err := chatService.ListRecent(ctx, host.User.ID, instantSlot.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(instantMessages) != 1 || instantMessages[0].Kind != chat.KindSystem ||
		instantMessages[0].SystemEventType == nil || *instantMessages[0].SystemEventType != chat.SystemEventMemberJoined ||
		instantMessages[0].Subject == nil || instantMessages[0].Subject.ID != instantJoiner.User.ID {
		t.Fatalf("expected one MEMBER_JOINED for instant joiner, got %#v", instantMessages)
	}

	waitlistMode := slot.AccessWaitlist
	waitlistDraft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Waitlist System Message", Activity: "coffee", PlaceText: "Center", Capacity: 2, AccessMode: &waitlistMode,
	}, "v11-chat-system-waitlist-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	waitlistSlot, err := slotService.PublishDraft(ctx, host.User.ID, waitlistDraft.ID, waitlistDraft.Version, "v11-chat-system-waitlist-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	// First two requesters auto-admit (capacity 2, filling both seats); the
	// third queues behind them.
	if _, err := slotService.Request(ctx, waitlistFirst.User.ID, waitlistSlot.ID, "v11-chat-system-waitlist-request-first-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, waitlistFiller.User.ID, waitlistSlot.ID, "v11-chat-system-waitlist-request-filler-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Request(ctx, waitlistSecond.User.ID, waitlistSlot.ID, "v11-chat-system-waitlist-request-second-0001"); err != nil {
		t.Fatal(err)
	}
	// The first member leaving frees a seat and promotes the queued one.
	if _, err := slotService.Leave(ctx, waitlistFirst.User.ID, waitlistSlot.ID, "v11-chat-system-waitlist-leave-0001"); err != nil {
		t.Fatal(err)
	}
	waitlistMessages, err := chatService.ListRecent(ctx, host.User.ID, waitlistSlot.ID, 100)
	if err != nil {
		t.Fatal(err)
	}
	wantWaitlist := []struct {
		event   chat.SystemEventType
		subject string
	}{
		{chat.SystemEventMemberJoined, waitlistFirst.User.ID},
		{chat.SystemEventMemberJoined, waitlistFiller.User.ID},
		{chat.SystemEventMemberLeft, waitlistFirst.User.ID},
		{chat.SystemEventMemberJoined, waitlistSecond.User.ID},
	}
	if len(waitlistMessages) != len(wantWaitlist) {
		t.Fatalf("expected %d SYSTEM messages, got %#v", len(wantWaitlist), waitlistMessages)
	}
	for i, want := range wantWaitlist {
		msg := waitlistMessages[i]
		if msg.Kind != chat.KindSystem || msg.SystemEventType == nil || *msg.SystemEventType != want.event || msg.Subject == nil || msg.Subject.ID != want.subject {
			t.Fatalf("waitlist message %d: want event=%s subject=%s, got %#v", i, want.event, want.subject, msg)
		}
	}
}

// TestV11ChatSystemMessageOnBlockRemoval covers the block-triggered
// MEMBER_LEFT path in block_store.go's bulk relationship-cleanup statement:
// a host blocking one accepted member must emit exactly one MEMBER_LEFT for
// that member, unaffected members keep full chat history including it, and
// the blocked member themselves is filtered out of chat entirely (README
// §8.1) rather than receiving the notice.
func TestV11ChatSystemMessageOnBlockRemoval(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "cbh", suffix)
	blocked := registerIntegrationUser(t, ctx, accountService, "cbb", suffix)
	stays := registerIntegrationUser(t, ctx, accountService, "cbs", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, blocked.User.ID, stays.User.ID}) })

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
	blockService, err := blocklist.NewService(NewBlockStore(pool, time.Hour))
	if err != nil {
		t.Fatal(err)
	}

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Block Removal", Activity: "coffee", PlaceText: "Center", Capacity: 2,
	}, "v11-chat-system-block-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	for _, member := range []string{blocked.User.ID, stays.User.ID} {
		if _, err := slotService.Request(ctx, member, created.ID, "v11-chat-system-block-request-"+member); err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member, "v11-chat-system-block-approve-"+member); err != nil {
			t.Fatal(err)
		}
	}

	if err := blockService.Block(ctx, host.User.ID, blocked.User.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = blockService.Unblock(ctx, host.User.ID, blocked.User.ID) })

	for _, viewer := range []string{host.User.ID, stays.User.ID} {
		messages, err := chatService.ListRecent(ctx, viewer, created.ID, 100)
		if err != nil {
			t.Fatalf("viewer %s: %v", viewer, err)
		}
		// Two MEMBER_JOINED (approve x2) followed by one MEMBER_LEFT for the
		// blocked member, and no second MEMBER_LEFT for the member who stays.
		if len(messages) != 3 {
			t.Fatalf("viewer %s: expected 3 SYSTEM messages, got %#v", viewer, messages)
		}
		left := messages[2]
		if left.Kind != chat.KindSystem || left.SystemEventType == nil || *left.SystemEventType != chat.SystemEventMemberLeft || left.Subject == nil || left.Subject.ID != blocked.User.ID {
			t.Fatalf("viewer %s: expected MEMBER_LEFT for blocked user, got %#v", viewer, left)
		}
	}

	if _, err := chatService.ListRecent(ctx, blocked.User.ID, created.ID, 100); !errors.Is(err, chat.ErrForbidden) {
		t.Fatalf("blocked member must not see the departure notice about themselves either, got %v", err)
	}
}
