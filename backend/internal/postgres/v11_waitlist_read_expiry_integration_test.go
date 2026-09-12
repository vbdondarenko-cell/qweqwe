package postgres

import (
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11WaitlistExpiredRequestReadsAsNone closes the read-side gap recorded
// in IMPLEMENTATION_STATUS.md §40: a WAITLIST slot_requests row that has
// expired (see promoteOldestWaitlistTx / waitlistRequest's TTL handling)
// must not keep reading back as the viewer's PENDING relationship on
// Get/ListPulse, and must drop out of the ListMine "REQUESTED" view, even
// though nothing has mutated the Slot since it expired (no background
// sweep exists; mutation-side cleanup only runs when something else next
// touches the queue).
func TestV11WaitlistExpiredRequestReadsAsNone(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "read-expiry-0001")

	// Fill both seats so the next requester queues instead of auto-admitting.
	for i := 0; i < 2; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, published.ID, "v11-waitlist-read-expiry-seat-"+users[i].User.ID); err != nil {
			t.Fatal(err)
		}
	}
	queued, err := service.Request(ctx, users[2].User.ID, published.ID, "v11-waitlist-read-expiry-queue-0001")
	if err != nil || queued.ViewerState != slot.ViewerPending {
		t.Fatalf("expected queued requester: viewer=%v err=%v", queued.ViewerState, err)
	}

	before, err := service.ListMine(ctx, users[2].User.ID, "REQUESTED")
	if err != nil {
		t.Fatal(err)
	}
	if !containsSlotWithViewer(before, published.ID, slot.ViewerPending) {
		t.Fatalf("live request should appear as PENDING in REQUESTED view: %#v", before)
	}

	backdateWaitlistRequest(t, ctx, pool, published.ID, users[2].User.ID, -2*time.Hour)

	// Get: the row is still physically present, but reads as NONE, not PENDING.
	got, err := service.Get(ctx, users[2].User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ViewerState != slot.ViewerNone {
		t.Fatalf("expired WAITLIST request must read as NONE, got %s", got.ViewerState)
	}

	// Pulse: same Slot, same viewer, same expectation.
	pulse, err := service.ListPulse(ctx, users[2].User.ID, "")
	if err != nil {
		t.Fatal(err)
	}
	if !containsSlotWithViewer(pulse, published.ID, slot.ViewerNone) {
		t.Fatalf("expired WAITLIST request must read as NONE in Pulse: %#v", pulse)
	}

	// ListMine REQUESTED: the Slot must no longer appear at all now that the
	// only thing that made it "requested" has expired.
	after, err := service.ListMine(ctx, users[2].User.ID, "REQUESTED")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range after {
		if s.ID == published.ID {
			t.Fatalf("expired WAITLIST request must not appear in REQUESTED view: %#v", after)
		}
	}

	// A live (non-expired) request from a different user on the same Slot is
	// unaffected by the other user's expiry.
	live, err := service.Request(ctx, users[3].User.ID, published.ID, "v11-waitlist-read-expiry-live-0001")
	if err != nil || live.ViewerState != slot.ViewerPending {
		t.Fatalf("unrelated live requester should still be PENDING: viewer=%v err=%v", live.ViewerState, err)
	}
	liveGet, err := service.Get(ctx, users[3].User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liveGet.ViewerState != slot.ViewerPending {
		t.Fatalf("live requester must still read PENDING, got %s", liveGet.ViewerState)
	}
}
