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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11LinksVisibilitySlot closes the discovery half of README §4.3's
// "Friends/Links" visibility mode, now that internal/friend (§55) exists:
// a LINKS Slot is invisible to a stranger on ListPulse/Get exactly like a
// PRIVATE one, becomes visible once a real, server-verified friendship
// exists (internal/friend, not just "the two users know each other" in
// some looser sense), and — proving the friendship check is not a
// standalone bypass around blocking — a block between the two still wins
// even after they are friends.
func TestV11LinksVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "lvh", suffix)
	other := registerIntegrationUser(t, ctx, accountService, "lvo", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, other.User.ID}) })

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
	friendStore, err := NewFriendStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	friendService, err := friend.NewService(friendStore)
	if err != nil {
		t.Fatal(err)
	}

	visibility := slot.VisibilityLinks
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Links Meetup", Activity: "coffee", PlaceText: "Center", Capacity: 3, Visibility: &visibility,
	}, "v11-links-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-links-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.Visibility != slot.VisibilityLinks || published.State != slot.StateFilling {
		t.Fatalf("published slot lost LINKS visibility or wrong state: %#v", published)
	}

	// (1) Not yet friends: ListPulse must exclude it, Get must 404.
	pulse, err := slotService.ListPulse(ctx, other.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range pulse {
		if item.ID == published.ID {
			t.Fatalf("LINKS slot must not appear in a non-friend's Pulse feed: %#v", item)
		}
	}
	if _, err := slotService.Get(ctx, other.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a non-friend reading a LINKS slot by ID, got %v", err)
	}

	// (2) Become real, server-verified friends.
	if _, err := friendService.Request(ctx, other.User.ID, host.User.ID); err != nil {
		t.Fatal(err)
	}
	if err := friendService.Accept(ctx, host.User.ID, other.User.ID); err != nil {
		t.Fatal(err)
	}

	pulseAfter, err := slotService.ListPulse(ctx, other.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range pulseAfter {
		if item.ID == published.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a friend to see the LINKS slot on Pulse: %#v", pulseAfter)
	}
	friendView, err := slotService.Get(ctx, other.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if friendView.ViewerState != slot.ViewerNone {
		t.Fatalf("a friend with no other relationship should read ViewerNone, got %s", friendView.ViewerState)
	}

	// (3) A block wins even between friends: not a bypass around blocking.
	if _, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, host.User.ID, other.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Get(ctx, other.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected a block to override LINKS friendship visibility, got %v", err)
	}
	pulseBlocked, err := slotService.ListPulse(ctx, other.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range pulseBlocked {
		if item.ID == published.ID {
			t.Fatalf("a blocked friend must not see the LINKS slot on Pulse: %#v", item)
		}
	}
}

// TestV11LinksVisibilityRequestReachableByDirectID proves LINKS keeps the
// same "discoverability gate, not access-control gate" contract §52
// established for PRIVATE: a stranger (not a friend) handed the Slot ID
// directly can still Request() it, because Request()/Join()/Approve()
// never reference visibility at all.
func TestV11LinksVisibilityRequestReachableByDirectID(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "lvr", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "lvs", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, stranger.User.ID}) })

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

	visibility := slot.VisibilityLinks
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Links Direct Request", Activity: "coffee", PlaceText: "Center", Capacity: 3, Visibility: &visibility,
	}, "v11-links-direct-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-links-direct-publish-0001")
	if err != nil {
		t.Fatal(err)
	}

	afterRequest, err := slotService.Request(ctx, stranger.User.ID, published.ID, "v11-links-direct-request-0001")
	if err != nil {
		t.Fatalf("a stranger with a known LINKS slot ID must still be able to Request(): %v", err)
	}
	if afterRequest.ViewerState != slot.ViewerPending {
		t.Fatalf("expected PENDING after Request() on a LINKS slot, got %s", afterRequest.ViewerState)
	}
}
