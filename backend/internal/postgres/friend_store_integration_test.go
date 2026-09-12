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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

type friendFixture struct {
	ctx        context.Context
	pool       *pgxpool.Pool
	store      *FriendStore
	capService *capability.Service
	a, b, c    account.AuthResult
}

func newFriendFixture(t *testing.T) *friendFixture {
	t.Helper()
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires LINKUP_TEST_DATABASE_URL and LINKUP_TEST_DATABASE_DESTRUCTIVE=1")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	t.Cleanup(cancel)
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
	a := registerIntegrationUser(t, ctx, accountService, "fa", suffix)
	b := registerIntegrationUser(t, ctx, accountService, "fb", suffix)
	c := registerIntegrationUser(t, ctx, accountService, "fc", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{a.User.ID, b.User.ID, c.User.ID}) })

	store, err := NewFriendStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	capStore, err := NewCapabilityStore(pool)
	if err != nil {
		t.Fatal(err)
	}
	capService, err := capability.NewService(capStore)
	if err != nil {
		t.Fatal(err)
	}

	return &friendFixture{ctx: ctx, pool: pool, store: store, capService: capService, a: a, b: b, c: c}
}

func TestFriendStoreRequestCreatesPendingAndNotifiesTarget(t *testing.T) {
	f := newFriendFixture(t)
	enableNotificationsForAll(t, f.ctx, f.pool)

	result, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != friend.OutcomeRequested {
		t.Fatalf("expected OutcomeRequested, got %s", result.Outcome)
	}

	outgoing, err := f.store.ListOutgoing(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoing) != 1 || outgoing[0].User.ID != f.b.User.ID {
		t.Fatalf("expected one outgoing request to b, got %#v", outgoing)
	}
	incoming, err := f.store.ListIncoming(f.ctx, f.b.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(incoming) != 1 || incoming[0].User.ID != f.a.User.ID {
		t.Fatalf("expected one incoming request from a, got %#v", incoming)
	}

	already, err := f.store.AreFriends(f.ctx, f.a.User.ID, f.b.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if already {
		t.Fatal("a PENDING request must not itself create a friendship")
	}

	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(f.pool, f.capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if processed := drainProjector(t, f.ctx, projector, 200); processed == 0 {
		t.Fatal("expected the friend.requested event to be processed")
	}
	targetCalls := recorder.callsFor(f.b.User.ID)
	if len(targetCalls) != 1 || targetCalls[0].message.Title != "New friend request" {
		t.Fatalf("expected target to receive one New friend request push, got %#v", targetCalls)
	}
	assertNotificationRow(t, f.ctx, f.pool, f.b.User.ID, "FRIEND_REQUEST", "SENT", func(deepLink string) bool {
		return deepLink == "app://me/requests"
	})
}

func TestFriendStoreReverseRequestAutoAcceptsAndNotifiesOriginalRequester(t *testing.T) {
	f := newFriendFixture(t)
	enableNotificationsForAll(t, f.ctx, f.pool)

	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	// b requests a back, while a's request to b is still PENDING: this must
	// resolve as an immediate mutual match, not a second parallel request.
	result, err := f.store.Request(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != friend.OutcomeAccepted {
		t.Fatalf("expected OutcomeAccepted for the reverse request, got %s", result.Outcome)
	}

	areFriends, err := f.store.AreFriends(f.ctx, f.a.User.ID, f.b.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !areFriends {
		t.Fatal("expected a and b to be friends after the reverse request auto-accepted")
	}
	// The reverse order must read identically.
	areFriendsReverse, err := f.store.AreFriends(f.ctx, f.b.User.ID, f.a.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !areFriendsReverse {
		t.Fatal("AreFriends must be symmetric")
	}

	// No pending requests remain in either direction.
	outgoingA, err := f.store.ListOutgoing(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoingA) != 0 {
		t.Fatalf("expected no remaining outgoing requests for a, got %#v", outgoingA)
	}
	incomingA, err := f.store.ListIncoming(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(incomingA) != 0 {
		t.Fatalf("expected no remaining incoming requests for a, got %#v", incomingA)
	}

	// The original requester (a) is the one who gets FRIEND_ACCEPTED, not b
	// (who just completed the match by requesting back).
	recorder := &recordingPusher{}
	projector, err := NewNotificationProjector(f.pool, f.capService, recorder, 14*24*time.Hour, 24*time.Hour, 5, 5*time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	drainProjector(t, f.ctx, projector, 200)
	aCalls := recorder.callsFor(f.a.User.ID)
	if len(aCalls) != 1 || aCalls[0].message.Title != "Friend request accepted" {
		t.Fatalf("expected a to receive one Friend request accepted push, got %#v", aCalls)
	}
	// b legitimately receives ONE push from this whole flow: the original
	// "New friend request" for a's first ask (a real, correct notification
	// that happened before b's reverse request auto-accepted it) — but
	// never a second "Friend request accepted" push for their own accept
	// action, which they already know happened.
	bCalls := recorder.callsFor(f.b.User.ID)
	if len(bCalls) != 1 || bCalls[0].message.Title != "New friend request" {
		t.Fatalf("expected b to receive exactly the original New friend request push, got %#v", bCalls)
	}
	for _, call := range bCalls {
		if call.message.Title == "Friend request accepted" {
			t.Fatalf("b must not be notified of their own accept action, got %#v", bCalls)
		}
	}
	assertNotificationRow(t, f.ctx, f.pool, f.a.User.ID, "FRIEND_ACCEPTED", "SENT", func(deepLink string) bool {
		return deepLink == "app://me/connections"
	})
}

func TestFriendStoreDuplicateForwardRequestIsIdempotent(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	result, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}
	if result.Outcome != friend.OutcomeAlreadyPending {
		t.Fatalf("expected OutcomeAlreadyPending on repeat request, got %s", result.Outcome)
	}
	outgoing, err := f.store.ListOutgoing(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoing) != 1 {
		t.Fatalf("expected exactly one outgoing request row despite the repeat call, got %d", len(outgoing))
	}
}

func TestFriendStoreRequestRejectsBlockedPair(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.pool.Exec(f.ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, f.a.User.ID, f.b.User.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); !errors.Is(err, friend.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a blocked pair, got %v", err)
	}
	// The reverse direction (blocked user requesting the blocker) is also
	// rejected — blocking is symmetric throughout this codebase.
	if _, err := f.store.Request(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); !errors.Is(err, friend.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for the reverse direction of a blocked pair, got %v", err)
	}
}

func TestFriendStoreRequestRejectsAlreadyFriends(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Accept(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); !errors.Is(err, friend.ErrAlreadyFriends) {
		t.Fatalf("expected ErrAlreadyFriends, got %v", err)
	}
}

func TestFriendStoreAcceptCreatesFriendshipAndFailsWhenNoPendingRequest(t *testing.T) {
	f := newFriendFixture(t)
	if err := f.store.Accept(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); !errors.Is(err, friend.ErrNotFound) {
		t.Fatalf("expected ErrNotFound with no pending request, got %v", err)
	}

	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Accept(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	areFriends, err := f.store.AreFriends(f.ctx, f.a.User.ID, f.b.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !areFriends {
		t.Fatal("expected a friendship after Accept")
	}

	// Idempotent replay: accepting again (e.g. a retried request) succeeds
	// silently instead of surfacing ErrNotFound for an already-answered row.
	if err := f.store.Accept(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); err != nil {
		t.Fatalf("expected idempotent replay of Accept to succeed, got %v", err)
	}
}

func TestFriendStoreRejectMarksRejectedAndIsIdempotent(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Reject(f.ctx, f.b.User.ID, f.a.User.ID); err != nil {
		t.Fatal(err)
	}
	incoming, err := f.store.ListIncoming(f.ctx, f.b.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(incoming) != 0 {
		t.Fatalf("expected no incoming requests after reject, got %#v", incoming)
	}
	// Idempotent replay.
	if err := f.store.Reject(f.ctx, f.b.User.ID, f.a.User.ID); err != nil {
		t.Fatalf("expected idempotent replay of Reject to succeed, got %v", err)
	}
	// Rejecting a request that never existed at all is a real error.
	if err := f.store.Reject(f.ctx, f.c.User.ID, f.a.User.ID); !errors.Is(err, friend.ErrNotFound) {
		t.Fatalf("expected ErrNotFound rejecting a request that never existed, got %v", err)
	}
}

func TestFriendStoreCancelMarksCancelledAndIsIdempotent(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Cancel(f.ctx, f.a.User.ID, f.b.User.ID); err != nil {
		t.Fatal(err)
	}
	outgoing, err := f.store.ListOutgoing(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(outgoing) != 0 {
		t.Fatalf("expected no outgoing requests after cancel, got %#v", outgoing)
	}
	if err := f.store.Cancel(f.ctx, f.a.User.ID, f.b.User.ID); err != nil {
		t.Fatalf("expected idempotent replay of Cancel to succeed, got %v", err)
	}
	if err := f.store.Cancel(f.ctx, f.c.User.ID, f.a.User.ID); !errors.Is(err, friend.ErrNotFound) {
		t.Fatalf("expected ErrNotFound cancelling a request that never existed, got %v", err)
	}
}

func TestFriendStoreRemoveIsIdempotentAndListFriendsReflectsIt(t *testing.T) {
	f := newFriendFixture(t)
	if _, err := f.store.Request(f.ctx, f.a.User.ID, f.b.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := f.store.Accept(f.ctx, f.b.User.ID, f.a.User.ID, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	friendsOfA, err := f.store.ListFriends(f.ctx, f.a.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(friendsOfA) != 1 || friendsOfA[0].User.ID != f.b.User.ID {
		t.Fatalf("expected a to have exactly one friend (b), got %#v", friendsOfA)
	}

	// Remove from either side must work regardless of who initiated.
	if err := f.store.Remove(f.ctx, f.b.User.ID, f.a.User.ID); err != nil {
		t.Fatal(err)
	}
	stillFriends, err := f.store.AreFriends(f.ctx, f.a.User.ID, f.b.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stillFriends {
		t.Fatal("expected the friendship to be removed")
	}
	// Idempotent: removing again is a harmless no-op, not an error.
	if err := f.store.Remove(f.ctx, f.a.User.ID, f.b.User.ID); err != nil {
		t.Fatalf("expected idempotent Remove to succeed, got %v", err)
	}
}
