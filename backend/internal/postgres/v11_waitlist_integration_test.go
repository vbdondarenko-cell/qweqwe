package postgres

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/blocklist"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func newV11WaitlistFixture(t *testing.T) (context.Context, *pgxpool.Pool, *slot.Service, account.AuthResult, []account.AuthResult) {
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
	accounts, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accounts, "wh", suffix)
	users := make([]account.AuthResult, 6)
	for i := range users {
		users[i] = registerIntegrationUser(t, ctx, accounts, fmt.Sprintf("w%d", i), suffix)
	}
	base, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewV11SlotStore(base)
	if err != nil {
		t.Fatal(err)
	}
	service, err := slot.NewService(store)
	if err != nil {
		t.Fatal(err)
	}
	return ctx, pool, service, host, users
}

func createPublishedWaitlist(t *testing.T, ctx context.Context, service *slot.Service, hostID string, capacity int, suffix string) slot.Slot {
	t.Helper()
	mode := slot.AccessWaitlist
	draft, err := service.CreateDraft(ctx, hostID, slot.CreateInput{
		Title: "Waitlist " + suffix, Activity: "coffee", PlaceText: "Center", Capacity: capacity, AccessMode: &mode,
	}, "v11-waitlist-draft-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishDraft(ctx, hostID, draft.ID, draft.Version, "v11-waitlist-publish-"+suffix)
	if err != nil {
		t.Fatal(err)
	}
	if published.AccessMode != slot.AccessWaitlist || published.State != slot.StateFilling {
		t.Fatalf("waitlist did not publish: %#v", published)
	}
	return published
}

func TestV11WaitlistFIFOAndSeatTransitionsIntegration(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "fifo-0001")

	for i := 0; i < 2; i++ {
		out, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-admit-%04d", i))
		if err != nil {
			t.Fatal(err)
		}
		if out.ViewerState != slot.ViewerAccepted {
			t.Fatalf("seat %d should auto-admit, got %s", i, out.ViewerState)
		}
	}
	for i := 2; i < 4; i++ {
		out, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-queue-%04d", i))
		if err != nil {
			t.Fatal(err)
		}
		if out.ViewerState != slot.ViewerPending {
			t.Fatalf("overflow %d should queue, got %s", i, out.ViewerState)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE slot_requests SET created_at=$3 WHERE slot_id=$1 AND user_id=$2`, published.ID, users[2].User.ID, time.Now().UTC().Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE slot_requests SET created_at=$3 WHERE slot_id=$1 AND user_id=$2`, published.ID, users[3].User.ID, time.Now().UTC().Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	if _, err := service.Leave(ctx, users[0].User.ID, published.ID, "v11-waitlist-leave-promote-0001"); err != nil {
		t.Fatal(err)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[2].User.ID, true, false)
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[3].User.ID, false, true)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 2, 2, string(slot.StateFull))

	hostView, err := service.Get(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RemoveMember(ctx, host.User.ID, published.ID, users[1].User.ID, hostView.Version, "v11-waitlist-kick-promote-0001"); err != nil {
		t.Fatal(err)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[3].User.ID, true, false)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 2, 2, string(slot.StateFull))

	queued, err := service.Request(ctx, users[4].User.ID, published.ID, "v11-waitlist-queue-capacity-0001")
	if err != nil || queued.ViewerState != slot.ViewerPending {
		t.Fatalf("expected queued user before capacity expansion: viewer=%s err=%v", queued.ViewerState, err)
	}
	hostView, err = service.Get(ctx, host.User.ID, published.ID)
	if err != nil {
		t.Fatal(err)
	}
	newCapacity := 3
	expanded, err := service.Edit(ctx, host.User.ID, published.ID, slot.EditInput{ExpectedVersion: hostView.Version, Capacity: &newCapacity}, "v11-waitlist-capacity-expand-0001")
	if err != nil {
		t.Fatal(err)
	}
	if expanded.Capacity != 3 || expanded.AcceptedCount != 3 || expanded.State != slot.StateFull {
		t.Fatalf("capacity expansion did not promote queue: %#v", expanded)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[4].User.ID, true, false)

	queued, err = service.Request(ctx, users[5].User.ID, published.ID, "v11-waitlist-withdraw-queue-0001")
	if err != nil || queued.ViewerState != slot.ViewerPending {
		t.Fatalf("withdraw candidate was not queued: viewer=%s err=%v", queued.ViewerState, err)
	}
	if _, err := service.Leave(ctx, users[5].User.ID, published.ID, "v11-waitlist-withdraw-0001"); err != nil {
		t.Fatal(err)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[5].User.ID, false, false)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 3, 3, string(slot.StateFull))
}

func TestV11WaitlistFullReopenRaceIntegration(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "race-0001")
	for i := 0; i < 3; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-race-seed-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE slot_requests SET created_at=$3 WHERE slot_id=$1 AND user_id=$2`, published.ID, users[2].User.ID, time.Now().UTC().Add(-2*time.Minute)); err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := service.Leave(context.Background(), users[0].User.ID, published.ID, "v11-waitlist-race-leave-0001")
		errs <- err
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := service.Request(context.Background(), users[3].User.ID, published.ID, "v11-waitlist-race-request-0001")
		errs <- err
	}()
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent waitlist mutation failed: %v", err)
		}
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[2].User.ID, true, false)
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[3].User.ID, false, true)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 2, 2, string(slot.StateFull))
}

func TestV11WaitlistBlockAcceptedMemberPromotesNextIntegration(t *testing.T) {
	ctx, pool, service, host, users := newV11WaitlistFixture(t)
	published := createPublishedWaitlist(t, ctx, service, host.User.ID, 2, "block-0001")
	for i := 0; i < 3; i++ {
		if _, err := service.Request(ctx, users[i].User.ID, published.ID, fmt.Sprintf("v11-waitlist-block-seed-%04d", i)); err != nil {
			t.Fatal(err)
		}
	}
	blocks, err := blocklist.NewService(NewBlockStore(pool))
	if err != nil {
		t.Fatal(err)
	}
	if err := blocks.Block(ctx, host.User.ID, users[0].User.ID); err != nil {
		t.Fatal(err)
	}
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[0].User.ID, false, false)
	assertWaitlistRelationship(t, ctx, pool, published.ID, users[2].User.ID, true, false)
	assertSlotCapacityInvariant(t, ctx, pool, published.ID, 2, 2, string(slot.StateFull))
}

func assertWaitlistRelationship(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID, userID string, member, pending bool) {
	t.Helper()
	var gotMember, gotPending bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM slot_memberships WHERE slot_id=$1 AND user_id=$2), EXISTS(SELECT 1 FROM slot_requests WHERE slot_id=$1 AND user_id=$2)`, slotID, userID).Scan(&gotMember, &gotPending); err != nil {
		t.Fatal(err)
	}
	if gotMember != member || gotPending != pending {
		t.Fatalf("relationship user=%s member=%v/%v pending=%v/%v", userID, gotMember, member, gotPending, pending)
	}
}

func assertSlotCapacityInvariant(t *testing.T, ctx context.Context, pool *pgxpool.Pool, slotID string, accepted, memberships int, state string) {
	t.Helper()
	var gotAccepted, gotMemberships int
	var gotState string
	if err := pool.QueryRow(ctx, `SELECT accepted_count,state FROM slots WHERE id=$1`, slotID).Scan(&gotAccepted, &gotState); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM slot_memberships WHERE slot_id=$1`, slotID).Scan(&gotMemberships); err != nil {
		t.Fatal(err)
	}
	if gotAccepted != accepted || gotMemberships != memberships || gotState != state {
		t.Fatalf("capacity invariant accepted=%d/%d memberships=%d/%d state=%s/%s", gotAccepted, accepted, gotMemberships, memberships, gotState, state)
	}
}
