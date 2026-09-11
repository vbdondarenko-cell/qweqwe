package postgres

import (
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11ListPendingWaitlistQueuePositionAndExpiry closes README §6.6's
// "complete roster/request/waitlist states": a host viewing the pending
// list for a WAITLIST Slot previously saw the exact same shape as an
// APPROVAL Slot (just user + requestedAt), with no way to see FIFO queue
// position or which positions are already stale per the same TTL every
// other WAITLIST-aware read (Get/ListPulse/ListMine/Map/realtime
// visibility) already applies.
func TestV11ListPendingWaitlistQueuePositionAndExpiry(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "pending-queue-0001")

	// Fill both seats so the next three requesters queue in order.
	for i := 0; i < 2; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, published.ID, "v11-pending-queue-seat-"+users[i].User.ID); err != nil {
			t.Fatal(err)
		}
	}
	var queued []string
	for i := 2; i < 5; i++ {
		r, err := service.Request(ctx, users[i].User.ID, published.ID, "v11-pending-queue-q-"+users[i].User.ID)
		if err != nil || r.ViewerState != slot.ViewerPending {
			t.Fatalf("expected queued requester %d: viewer=%v err=%v", i, r.ViewerState, err)
		}
		queued = append(queued, users[i].User.ID)
		// Ensure distinct created_at ordering even if the host clock has
		// coarse resolution in this environment.
		time.Sleep(2 * time.Millisecond)
	}

	// Backdate the oldest (first) position past the TTL. This is the only
	// realistic scenario: FIFO order is entirely a function of created_at,
	// so the position that expires first, in reality, is always whichever
	// has been queued longest — backdating any other position would also
	// move it ahead of the others in order, which is a test-harness
	// artifact of rewriting created_at, not something that can happen from
	// time simply passing.
	backdateWaitlistRequest(t, ctx, pool, published.ID, queued[0], -2*time.Hour)

	pending, err := service.ListPending(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending WAITLIST requests, got %d: %#v", len(pending), pending)
	}
	for i, want := range queued {
		got := pending[i]
		if got.User.ID != want {
			t.Fatalf("position %d: expected user %s, got %s (order must be FIFO by created_at): %#v", i, want, got.User.ID, pending)
		}
		if got.QueuePosition == nil || *got.QueuePosition != i+1 {
			t.Fatalf("position %d: expected QueuePosition=%d, got %#v", i, i+1, got.QueuePosition)
		}
		wantExpired := i == 0
		if got.Expired != wantExpired {
			t.Fatalf("position %d (user %s): expected Expired=%v, got %v", i, want, wantExpired, got.Expired)
		}
	}
}

// TestV11ListPendingApprovalModeHasNoQueueFields is the regression check:
// APPROVAL mode has no queue or expiry concept, so QueuePosition must stay
// nil and Expired must stay false for every entry, exactly matching v1.0's
// original contract.
func TestV11ListPendingApprovalModeHasNoQueueFields(t *testing.T) {
	ctx, _, service, host, users := newV11WaitlistFixture(t)

	created, err := service.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Approval Pending Fields", Activity: "coffee", PlaceText: "Center", Capacity: 3,
	}, "v11-pending-approval-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, created.ID, "v11-pending-approval-request-"+users[i].User.ID); err != nil {
			t.Fatal(err)
		}
	}

	pending, err := service.ListPending(ctx, host.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending APPROVAL requests, got %d", len(pending))
	}
	for _, item := range pending {
		if item.QueuePosition != nil {
			t.Fatalf("APPROVAL mode must never report a QueuePosition, got %#v", item)
		}
		if item.Expired {
			t.Fatalf("APPROVAL mode must never report Expired=true, got %#v", item)
		}
	}
}
