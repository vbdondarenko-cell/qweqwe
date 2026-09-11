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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11SelectedVisibilitySlot closes the discovery half of README §4.3's
// "Selected people" mode: a SELECTED Slot is invisible to a stranger on
// ListPulse/Get, visible to whichever specific users the host put on the
// allow-list at creation time, and a block still wins even over being
// selected — proving the allow-list is not a standalone bypass around
// blocking, matching the same invariant already proven for LINKS.
func TestV11SelectedVisibilitySlot(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "svh", suffix)
	selected := registerIntegrationUser(t, ctx, accountService, "svs", suffix)
	other := registerIntegrationUser(t, ctx, accountService, "svo", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, selected.User.ID, other.User.ID}) })

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

	visibility := slot.VisibilitySelected
	draft, err := slotService.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Selected Meetup", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, SelectedUserIDs: []string{selected.User.ID},
	}, "v11-selected-visibility-draft-0001")
	if err != nil {
		t.Fatal(err)
	}
	if draft.Visibility != slot.VisibilitySelected || len(draft.SelectedUserIDs) != 1 || draft.SelectedUserIDs[0] != selected.User.ID {
		t.Fatalf("draft did not preserve SELECTED visibility/allow-list: %#v", draft)
	}
	published, err := slotService.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "v11-selected-visibility-publish-0001")
	if err != nil {
		t.Fatal(err)
	}
	if published.Visibility != slot.VisibilitySelected || published.State != slot.StateFilling {
		t.Fatalf("published slot lost SELECTED visibility or wrong state: %#v", published)
	}

	// (1) A non-selected stranger: excluded from ListPulse, Get 404s.
	pulse, err := slotService.ListPulse(ctx, other.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range pulse {
		if item.ID == published.ID {
			t.Fatalf("SELECTED slot must not appear in a non-selected stranger's Pulse feed: %#v", item)
		}
	}
	if _, err := slotService.Get(ctx, other.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected ErrNotFound for a non-selected stranger reading a SELECTED slot by ID, got %v", err)
	}

	// (2) The selected user: included in ListPulse, Get succeeds.
	pulseSelected, err := slotService.ListPulse(ctx, selected.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, item := range pulseSelected {
		if item.ID == published.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the selected user to see the SELECTED slot on Pulse: %#v", pulseSelected)
	}
	selectedView, err := slotService.Get(ctx, selected.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if selectedView.ViewerState != slot.ViewerNone {
		t.Fatalf("a selected user with no other relationship should read ViewerNone, got %s", selectedView.ViewerState)
	}

	// (3) A block wins even for a selected user.
	if _, err := pool.Exec(ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, host.User.ID, selected.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Get(ctx, selected.User.ID, published.ID); !errors.Is(err, slot.ErrNotFound) {
		t.Fatalf("expected a block to override SELECTED allow-list visibility, got %v", err)
	}

	// (4) A non-selected stranger with the Slot ID directly can still
	// Request() it — same "discoverability gate, not access-control gate"
	// contract PRIVATE/LINKS already established.
	afterRequest, err := slotService.Request(ctx, other.User.ID, published.ID, "v11-selected-visibility-request-0001")
	if err != nil {
		t.Fatalf("a non-selected stranger with a known SELECTED slot ID must still be able to Request(): %v", err)
	}
	if afterRequest.ViewerState != slot.ViewerPending {
		t.Fatalf("expected PENDING after Request() on a SELECTED slot, got %s", afterRequest.ViewerState)
	}
}

// TestV11SelectedVisibilityRejectedForLegacyCreate is the regression check
// for the v1.0 create endpoint, mirroring PRIVATE/LINKS's own.
func TestV11SelectedVisibilityRejectedForLegacyCreate(t *testing.T) {
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
	host := registerIntegrationUser(t, ctx, accountService, "svl", suffix)
	other := registerIntegrationUser(t, ctx, accountService, "svm", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, other.User.ID}) })

	store, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	visibility := slot.VisibilitySelected
	if _, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Legacy Selected Attempt", Activity: "coffee", PlaceText: "Center", Capacity: 3,
		Visibility: &visibility, SelectedUserIDs: []string{other.User.ID},
	}, "v11-selected-visibility-legacy-0001"); !errors.Is(err, slot.ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState rejecting SELECTED on the v1.0 create endpoint, got %v", err)
	}
}
