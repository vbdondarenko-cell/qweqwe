package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// README §6.6 "request expiry/withdrawal hardening": an expired WAITLIST
// queue position must never be promoted, must be purged as soon as the
// promotion path encounters it (no background sweep exists), and the
// requesting user must be able to request again once their own position
// has expired instead of hitting a confusing duplicate-request error.
// These tests fix `created_at` directly (rather than sleeping) so expiry
// is exercised deterministically against the fixture's 1h TTL.

func TestV11WaitlistExpiredRequestSkippedAndPurgedOnPromotion(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "expiry-promote-0001")

	// users[0] and users[1] fill both seats immediately.
	for i := 0; i < 2; i++ {
		if out, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-expiry-seat-%04d", i)); err != nil || out.ViewerState != slot.ViewerAccepted {
			t.Fatalf("seat %d did not auto-admit: viewer=%v err=%v", i, out.ViewerState, err)
		}
	}
	// users[2] and users[3] queue behind it, users[2] first.
	if out, err := service.Request(ctx, users[2].User.ID, published.ID, "v11-waitlist-expiry-queue-0001"); err != nil || out.ViewerState != slot.ViewerPending {
		t.Fatalf("users[2] should queue: viewer=%v err=%v", out.ViewerState, err)
	}
	if out, err := service.Request(ctx, users[3].User.ID, published.ID, "v11-waitlist-expiry-queue-0002"); err != nil || out.ViewerState != slot.ViewerPending {
		t.Fatalf("users[3] should queue: viewer=%v err=%v", out.ViewerState, err)
	}

	// Backdate users[2]'s queue position past the fixture's 1h TTL. users[3]
	// stays fresh and is behind users[2] in strict FIFO order.
	backdateWaitlistRequest(t, ctx, pool, published.ID, users[2].User.ID, -2*time.Hour)

	// Freeing a seat must skip+purge the expired users[2] and promote the
	// still-eligible users[3] instead, even though users[3] queued later.
	if _, err := service.Leave(ctx, users[0].User.ID, published.ID, "v11-waitlist-expiry-leave-0001"); err != nil {
		t.Fatal(err)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[2].User.ID, false, false)
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[3].User.ID, true, false)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 2, 2, string(slot.StateFull))
}

func TestV11WaitlistRequestAfterOwnExpiryIsAllowedNotDuplicate(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "expiry-rerequest-0001")

	// Fill both seats so the next requester queues instead of auto-admitting.
	for i := 0; i < 2; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-expiry-rerequest-seat-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	out, err := service.Request(ctx, users[2].User.ID, published.ID, "v11-waitlist-expiry-rerequest-first-0001")
	if err != nil || out.ViewerState != slot.ViewerPending {
		t.Fatalf("first request should queue: viewer=%v err=%v", out.ViewerState, err)
	}

	// A live (non-expired) duplicate request is still rejected.
	if _, err := service.Request(ctx, users[2].User.ID, published.ID, "v11-waitlist-expiry-rerequest-live-dup-0001"); err != slot.ErrDuplicateRequest {
		t.Fatalf("expected duplicate request rejection while still live, got %v", err)
	}

	backdateWaitlistRequest(t, ctx, pool, published.ID, users[2].User.ID, -2*time.Hour)

	// Once expired, the same user can request again instead of being stuck
	// behind ErrDuplicateRequest on a queue position that will never promote.
	again, err := service.Request(ctx, users[2].User.ID, published.ID, "v11-waitlist-expiry-rerequest-second-0001")
	if err != nil {
		t.Fatalf("re-request after own expiry should succeed: %v", err)
	}
	if again.ViewerState != slot.ViewerPending {
		t.Fatalf("re-request should still queue behind the full seat: %#v", again)
	}
	var requestedAt time.Time
	if err := pool.QueryRow(ctx, `SELECT created_at FROM slot_requests WHERE slot_id=$1 AND user_id=$2`, published.ID, users[2].User.ID).Scan(&requestedAt); err != nil {
		t.Fatal(err)
	}
	if time.Since(requestedAt) > time.Minute {
		t.Fatalf("re-request did not replace the stale queue position: created_at=%s", requestedAt)
	}
}

func backdateWaitlistRequest(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID, userID string, age time.Duration) {
	t.Helper()
	if _, err := pool.Exec(ctx, `UPDATE slot_requests SET created_at=$3 WHERE slot_id=$1 AND user_id=$2`, slotID, userID, time.Now().UTC().Add(age)); err != nil {
		t.Fatal(err)
	}
}
