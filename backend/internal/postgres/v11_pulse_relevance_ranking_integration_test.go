package postgres

import (
	"context"
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

// TestV11PulseRelevanceSortRanksInterestMatchAheadOfRecency proves README
// §6.10's Recommendation/ranking pipeline "Layer 2 — deterministic ranking
// from explicit user signals" actually works end to end against the real
// database: a Slot whose activity matches one of the viewer's own declared
// interests (account.ProfilePatch.Interests, set via PATCH /v1/me) ranks
// ahead of a strictly newer Slot that doesn't, when the viewer asks for
// GET /v1/pulse?sort=relevance -- while the long-standing recency default
// (sort omitted, or sort=recency) is completely unaffected and still
// returns pure created_at DESC.
func TestV11PulseRelevanceSortRanksInterestMatchAheadOfRecency(t *testing.T) {
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
	assertMigrationCount(t, ctx, pool, 37)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "prh", suffix)
	viewer := registerIntegrationUser(t, ctx, accountService, "prv", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{host.User.ID, viewer.User.ID}) })

	interests := []string{"board games"}
	if _, err := accountService.UpdateProfile(ctx, viewer.User.ID, account.ProfilePatch{Interests: &interests}); err != nil {
		t.Fatalf("set viewer interests: %v", err)
	}

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

	// The non-matching Slot is created second, so it is strictly newer --
	// under RECENCY sort it must rank first. Under RELEVANCE sort the
	// older, interest-matching Slot must rank first instead.
	matching, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Board Games Night", Activity: "board games", PlaceText: "Cafe Central", Capacity: 4,
	}, "v11-pulse-relevance-matching-0001")
	if err != nil {
		t.Fatalf("create matching slot: %v", err)
	}
	nonMatching, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Chess Meetup", Activity: "chess", PlaceText: "Library", Capacity: 4,
	}, "v11-pulse-relevance-nonmatching-0001")
	if err != nil {
		t.Fatalf("create non-matching slot: %v", err)
	}

	recency, err := slotService.ListPulse(ctx, viewer.User.ID, "")
	if err != nil {
		t.Fatalf("ListPulse recency: %v", err)
	}
	if idx := firstIndexOfSlot(recency, nonMatching.ID, matching.ID); idx != nonMatching.ID {
		t.Fatalf("recency default must rank the newer Slot first regardless of interests, got order %#v", slotIDs(recency))
	}

	relevance, err := slotService.ListPulse(ctx, viewer.User.ID, "relevance")
	if err != nil {
		t.Fatalf("ListPulse relevance: %v", err)
	}
	if idx := firstIndexOfSlot(relevance, nonMatching.ID, matching.ID); idx != matching.ID {
		t.Fatalf("relevance sort must rank the interest-matching Slot first even though it is older, got order %#v", slotIDs(relevance))
	}

	// Case-insensitivity: RELEVANCE is also a valid value spelled any way.
	if _, err := slotService.ListPulse(ctx, viewer.User.ID, "ReLeVaNcE"); err != nil {
		t.Fatalf("ListPulse relevance (mixed case): %v", err)
	}
}

// firstIndexOfSlot returns whichever of a/b appears first in items, to
// assert relative order without depending on exactly which other fixture
// rows (if any) happen to also be present.
func firstIndexOfSlot(items []slot.Slot, a, b string) string {
	for _, item := range items {
		if item.ID == a {
			return a
		}
		if item.ID == b {
			return b
		}
	}
	return ""
}

func slotIDs(items []slot.Slot) []string {
	ids := make([]string, len(items))
	for i, item := range items {
		ids[i] = item.ID
	}
	return ids
}
