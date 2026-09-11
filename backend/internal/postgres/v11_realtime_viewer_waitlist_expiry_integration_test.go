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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11RealtimeViewerWaitlistExpiryVisibility closes the
// realtime_viewer_store.go visibility-grant gap deliberately deferred
// across §44, §47 and §49: viewerRealtimeSQL's EXISTS(slot_requests)
// branch, which grants a viewer realtime visibility into a Slot's own
// lifecycle events (slot.created/updated/state_changed) because they have a
// WAITLIST queue position, never carried the same WAITLIST-TTL awareness
// the read side (Get/ListPulse/ListMine, §44) and the Map (PlaceSlots,
// §47) already have.
//
// Honest scope note: today this branch has no reachable production impact
// on its own — every currently-created Slot is PUBLIC (README §4.3: wider
// visibility is v1.1 scope not yet exposed), and a PUBLIC Slot in
// PUBLISHED/FILLING/FULL already grants blanket visibility to every viewer
// via viewerRealtimeSQL's separate PUBLIC branch, while every transition
// that moves a Slot out of that state range (Start, Cancel) unconditionally
// purges slot_requests first, so no requester — expired or not — can ever
// reach this branch while it would matter. The fix (and this test) exist
// for correctness and for the moment non-PUBLIC visibility ships, not
// because a live exposure was found; the test flips slots.visibility
// directly via SQL (the schema already allows it; no domain method sets it
// yet) to isolate and prove the branch's own logic ahead of that.
func TestV11RealtimeViewerWaitlistExpiryVisibility(t *testing.T) {
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
		t.Fatal(err)
	}

	accounts, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	v11Store, err := NewV11SlotStore(baseStore, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slots, err := slot.NewService(v11Store)
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
	host := registerIntegrationUser(t, ctx, accounts, "rwh", suffix)
	seatA := registerIntegrationUser(t, ctx, accounts, "rwa", suffix)
	seatB := registerIntegrationUser(t, ctx, accounts, "rwb", suffix)
	queued := registerIntegrationUser(t, ctx, accounts, "rwq", suffix)
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, seatA.User.ID, seatB.User.ID, queued.User.ID})
	})

	// Capacity must be >=2 (domain minimum); two seats are filled below so
	// the third requester queues instead of auto-admitting.
	waitlistMode := slot.AccessWaitlist
	draft, err := slots.CreateDraft(ctx, host.User.ID, slot.CreateInput{
		Title: "Realtime Waitlist Expiry", Activity: "coffee", PlaceText: "Center", Capacity: 2, AccessMode: &waitlistMode,
	}, "rt-waitlist-expiry-draft-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	published, err := slots.PublishDraft(ctx, host.User.ID, draft.ID, draft.Version, "rt-waitlist-expiry-publish-"+suffix)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := slots.Request(ctx, seatA.User.ID, published.ID, "rt-waitlist-expiry-seat-a-"+suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := slots.Request(ctx, seatB.User.ID, published.ID, "rt-waitlist-expiry-seat-b-"+suffix); err != nil {
		t.Fatal(err)
	}
	queuedResult, err := slots.Request(ctx, queued.User.ID, published.ID, "rt-waitlist-expiry-queue-"+suffix)
	if err != nil || queuedResult.ViewerState != slot.ViewerPending {
		t.Fatalf("expected queued requester: viewer=%v err=%v", queuedResult.ViewerState, err)
	}

	// The API only ever creates PUBLIC-visibility Slots today (README §4.3:
	// broader visibility is v1.1 scope not yet exposed), and a PUBLIC Slot
	// in FILLING/FULL already grants blanket realtime visibility to every
	// viewer via viewerRealtimeSQL's separate PUBLIC branch — which would
	// swamp and hide any effect from the slot_requests/WAITLIST branch this
	// test targets. The schema already supports non-PUBLIC values (see
	// migration 000002's CHECK constraint) even though no domain method
	// sets one yet, so this directly flips the column to isolate the
	// branch under test — legitimate coverage for behavior that becomes
	// live the moment non-PUBLIC visibility ships, not a fake scenario.
	if _, err := pool.Exec(ctx, `UPDATE slots SET visibility='PRIVATE' WHERE id=$1`, published.ID); err != nil {
		t.Fatal(err)
	}

	before := realtimeMaxSequence(t, ctx, pool)
	titleV1 := "Realtime Waitlist Expiry v1"
	editedV1, err := slots.Edit(ctx, host.User.ID, published.ID, slot.EditInput{
		ExpectedVersion: queuedResult.Version, Title: &titleV1,
	}, "rt-waitlist-expiry-edit1-"+suffix)
	if err != nil {
		t.Fatal(err)
	}

	// Before expiry: the queued (still-live) requester sees the update,
	// same as host and the accepted seat holder.
	beforeExpiryBatch, err := feed.Pull(ctx, queued.User.ID, before, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if !hasRealtimeSlotType(beforeExpiryBatch.Events, published.ID, "slot.updated") {
		t.Fatalf("live WAITLIST position should grant realtime visibility before expiry: %#v", beforeExpiryBatch.Events)
	}
	for _, viewer := range []account.AuthResult{host, seatA, seatB} {
		batch, err := feed.Pull(ctx, viewer.User.ID, before, realtime.MaxViewerBatch)
		if err != nil {
			t.Fatal(err)
		}
		if !hasRealtimeSlotType(batch.Events, published.ID, "slot.updated") {
			t.Fatalf("user %s should see the update regardless of WAITLIST expiry: %#v", viewer.User.ID, batch.Events)
		}
	}

	backdateWaitlistRequest(t, ctx, pool, published.ID, queued.User.ID, -2*time.Hour)

	afterExpiryStart := realtimeMaxSequence(t, ctx, pool)
	titleV2 := "Realtime Waitlist Expiry v2"
	if _, err := slots.Edit(ctx, host.User.ID, published.ID, slot.EditInput{
		ExpectedVersion: editedV1.Version, Title: &titleV2,
	}, "rt-waitlist-expiry-edit2-"+suffix); err != nil {
		t.Fatal(err)
	}

	// After expiry: the same viewer's now-stale WAITLIST position no
	// longer grants realtime visibility into this Slot's lifecycle events.
	afterExpiryBatch, err := feed.Pull(ctx, queued.User.ID, afterExpiryStart, realtime.MaxViewerBatch)
	if err != nil {
		t.Fatal(err)
	}
	if hasRealtimeSlotType(afterExpiryBatch.Events, published.ID, "slot.updated") {
		t.Fatalf("expired WAITLIST position must not grant realtime visibility: %#v", afterExpiryBatch.Events)
	}

	// Every other access path is completely unaffected by that one
	// viewer's expiry: host and the accepted seat holder still see it.
	for _, viewer := range []account.AuthResult{host, seatA, seatB} {
		batch, err := feed.Pull(ctx, viewer.User.ID, afterExpiryStart, realtime.MaxViewerBatch)
		if err != nil {
			t.Fatal(err)
		}
		if !hasRealtimeSlotType(batch.Events, published.ID, "slot.updated") {
			t.Fatalf("user %s must still see the update after another viewer's WAITLIST position expired: %#v", viewer.User.ID, batch.Events)
		}
	}

	// The cursor still advances across the now-hidden event for the
	// expired viewer, so pagination cannot get stuck (mirrors the
	// blocked-stranger assertion in TestV11RealtimeViewerFeedIntegration).
	if afterExpiryBatch.Cursor <= afterExpiryStart {
		t.Fatalf("cursor did not advance across the hidden event: before=%d after=%d", afterExpiryStart, afterExpiryBatch.Cursor)
	}
}
