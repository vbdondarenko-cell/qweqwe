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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type bumpFixture struct {
	ctx                       context.Context
	pool                      *pgxpool.Pool
	store                     *BumpStore
	slotService               *slot.Service
	host, memberA, memberB    account.AuthResult
	stranger                  account.AuthResult
	activeSlot, completedSlot slot.Slot
}

func newBumpFixture(t *testing.T) *bumpFixture {
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
	host := registerIntegrationUser(t, ctx, accountService, "bh", suffix)
	memberA := registerIntegrationUser(t, ctx, accountService, "ba", suffix)
	memberB := registerIntegrationUser(t, ctx, accountService, "bb", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "bs", suffix)
	t.Cleanup(func() {
		cleanupIntegrationRows(pool, []string{host.User.ID, memberA.User.ID, memberB.User.ID, stranger.User.ID})
	})

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}

	store, err := NewBumpStore(pool)
	if err != nil {
		t.Fatal(err)
	}

	// activeSlot: host + memberA + memberB accepted, then started -> ACTIVE.
	activeSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Bump Active", Activity: "coffee", PlaceText: "Center", Capacity: 3,
	}, "bump-active-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []account.AuthResult{memberA, memberB} {
		if _, err := slotService.Request(ctx, m.User.ID, activeSlot.ID, "bump-active-request-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Approve(ctx, host.User.ID, activeSlot.ID, m.User.ID, "bump-active-approve-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
	}
	activeSlot, err = slotService.Start(ctx, host.User.ID, activeSlot.ID, "bump-active-start-0001")
	if err != nil {
		t.Fatal(err)
	}
	if activeSlot.State != slot.StateActive {
		t.Fatalf("expected ACTIVE, got %s", activeSlot.State)
	}

	// completedSlot: same shape, but driven all the way to COMPLETED.
	completedSlot, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Bump Completed", Activity: "coffee", PlaceText: "Center", Capacity: 3,
	}, "bump-completed-create-0001")
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range []account.AuthResult{memberA, memberB} {
		if _, err := slotService.Request(ctx, m.User.ID, completedSlot.ID, "bump-completed-request-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
		if _, err := slotService.Approve(ctx, host.User.ID, completedSlot.ID, m.User.ID, "bump-completed-approve-"+m.User.ID); err != nil {
			t.Fatal(err)
		}
	}
	completedSlot, err = slotService.Start(ctx, host.User.ID, completedSlot.ID, "bump-completed-start-0001")
	if err != nil {
		t.Fatal(err)
	}
	completedSlot, err = slotService.Complete(ctx, host.User.ID, completedSlot.ID, "bump-completed-complete-0001")
	if err != nil {
		t.Fatal(err)
	}
	if completedSlot.State != slot.StateCompleted {
		t.Fatalf("expected COMPLETED, got %s", completedSlot.State)
	}

	return &bumpFixture{
		ctx: ctx, pool: pool, store: store, slotService: slotService,
		host: host, memberA: memberA, memberB: memberB, stranger: stranger,
		activeSlot: activeSlot, completedSlot: completedSlot,
	}
}

func TestBumpStoreIssueChallengeRequiresEligibleParticipant(t *testing.T) {
	f := newBumpFixture(t)

	if _, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute); err != nil {
		t.Fatalf("accepted member should be eligible on ACTIVE slot: %v", err)
	}
	if _, err := f.store.IssueChallenge(f.ctx, f.host.User.ID, f.completedSlot.ID, time.Minute); err != nil {
		t.Fatalf("host should be eligible on COMPLETED slot: %v", err)
	}
	if _, err := f.store.IssueChallenge(f.ctx, f.stranger.User.ID, f.activeSlot.ID, time.Minute); !errors.Is(err, bump.ErrNotEligible) {
		t.Fatalf("stranger must not be eligible, got %v", err)
	}
	if _, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, "00000000-0000-0000-0000-000000000000", time.Minute); !errors.Is(err, bump.ErrNotFound) {
		t.Fatalf("nonexistent slot must report ErrNotFound, got %v", err)
	}
}

func TestBumpStoreMutualConfirmVerifiesBothDirectionsOnly(t *testing.T) {
	f := newBumpFixture(t)

	chA, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// One direction: memberA claims memberB. Must NOT verify or credit
	// reliability yet -- "client tap alone can never increase trust".
	resultA, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, chA.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if resultA.Verified {
		t.Fatal("a single one-directional submission must not verify")
	}
	relA, err := f.store.Reliability(f.ctx, f.memberA.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if relA.VerifiedBumpCount != 0 || relA.Band != bump.BandNew {
		t.Fatalf("submitter's own reliability must not move before the pair is mutual: %#v", relA)
	}

	// Reverse direction: memberB claims memberA. Now the pair is mutual.
	chB, err := f.store.IssueChallenge(f.ctx, f.memberB.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	resultB, err := f.store.Confirm(f.ctx, f.memberB.User.ID, f.activeSlot.ID, f.memberA.User.ID, chB.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if !resultB.Verified {
		t.Fatal("the second (matching) direction must verify the pair")
	}

	for _, u := range []string{f.memberA.User.ID, f.memberB.User.ID} {
		rel, err := f.store.Reliability(f.ctx, u)
		if err != nil {
			t.Fatal(err)
		}
		if rel.VerifiedBumpCount != 1 || rel.Band != bump.BandBuilding {
			t.Fatalf("user %s: expected count=1 band=BUILDING after mutual confirm, got %#v", u, rel)
		}
	}

	vaultA, err := f.store.Vault(f.ctx, f.memberA.User.ID, 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(vaultA) != 1 || vaultA[0].SlotID != f.activeSlot.ID || vaultA[0].CounterpartID != f.memberB.User.ID {
		t.Fatalf("unexpected vault contents: %#v", vaultA)
	}
}

func TestBumpStoreConfirmRejectsInvalidOrReplayedChallenge(t *testing.T) {
	f := newBumpFixture(t)

	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, bump.ErrInvalidChallenge) {
		t.Fatalf("unknown nonce must be rejected, got %v", err)
	}

	ch, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	// Wrong actor for this nonce.
	if _, err := f.store.Confirm(f.ctx, f.memberB.User.ID, f.activeSlot.ID, f.memberA.User.ID, ch.Nonce); !errors.Is(err, bump.ErrInvalidChallenge) {
		t.Fatalf("nonce issued to a different user must be rejected, got %v", err)
	}

	// Correct first use succeeds (one direction only, does not verify).
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, ch.Nonce); err != nil {
		t.Fatal(err)
	}
	// Replay of the SAME nonce must fail: single-use.
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, ch.Nonce); !errors.Is(err, bump.ErrInvalidChallenge) {
		t.Fatalf("consumed nonce must be rejected on replay, got %v", err)
	}

	// An expired challenge (TTL already elapsed) must also be rejected.
	// The TTL must be at least ~1ms: Postgres timestamptz has only
	// microsecond precision, so a sub-microsecond TTL can round
	// issued_at/expires_at to the identical stored instant and trip the
	// migration's own `expires_at > issued_at` CHECK instead of exercising
	// expiry at all.
	expired, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, 5*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, expired.Nonce); !errors.Is(err, bump.ErrInvalidChallenge) {
		t.Fatalf("expired nonce must be rejected, got %v", err)
	}
}

func TestBumpStoreConfirmRejectsNonParticipantCounterpart(t *testing.T) {
	f := newBumpFixture(t)

	ch, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.stranger.User.ID, ch.Nonce); !errors.Is(err, bump.ErrNotEligible) {
		t.Fatalf("counterpart who was never in the Slot must be rejected, got %v", err)
	}
}

func TestBumpStoreConfirmRejectsBlockedCounterpart(t *testing.T) {
	f := newBumpFixture(t)

	if _, err := f.pool.Exec(f.ctx, `INSERT INTO user_blocks (blocker_id, blocked_id) VALUES ($1,$2)`, f.memberA.User.ID, f.memberB.User.ID); err != nil {
		t.Fatal(err)
	}
	ch, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, ch.Nonce); !errors.Is(err, bump.ErrForbidden) {
		t.Fatalf("blocked pair must be rejected, got %v", err)
	}
}

func TestBumpStoreOneEventOneContributionIsIdempotent(t *testing.T) {
	f := newBumpFixture(t)

	// memberA claims memberB twice (two separate challenges) on the same
	// Slot: the second claim must be a harmless no-op, not a duplicate
	// contribution, and must not itself trigger verification.
	ch1, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, ch1.Nonce); err != nil {
		t.Fatal(err)
	}
	ch2, err := f.store.IssueChallenge(f.ctx, f.memberA.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	result2, err := f.store.Confirm(f.ctx, f.memberA.User.ID, f.activeSlot.ID, f.memberB.User.ID, ch2.Nonce)
	if err != nil {
		t.Fatal(err)
	}
	if result2.Verified {
		t.Fatal("a repeat one-directional claim must not verify by itself")
	}

	// Now complete the mutual pair once from the other side.
	chB, err := f.store.IssueChallenge(f.ctx, f.memberB.User.ID, f.activeSlot.ID, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.store.Confirm(f.ctx, f.memberB.User.ID, f.activeSlot.ID, f.memberA.User.ID, chB.Nonce); err != nil {
		t.Fatal(err)
	}

	var submissionCount int
	if err := f.pool.QueryRow(f.ctx, `
		SELECT count(*) FROM bump_submissions WHERE slot_id=$1 AND submitter_id=$2 AND counterpart_id=$3`,
		f.activeSlot.ID, f.memberA.User.ID, f.memberB.User.ID).Scan(&submissionCount); err != nil {
		t.Fatal(err)
	}
	if submissionCount != 1 {
		t.Fatalf("duplicate directed submissions must not create multiple rows, got %d", submissionCount)
	}

	rel, err := f.store.Reliability(f.ctx, f.memberA.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rel.VerifiedBumpCount != 1 {
		t.Fatalf("repeat one-directional claims before mutual confirm must not inflate the count: got %d", rel.VerifiedBumpCount)
	}
}

func TestBumpStoreReliabilityDefaultsForUnknownUser(t *testing.T) {
	f := newBumpFixture(t)
	rel, err := f.store.Reliability(f.ctx, f.stranger.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rel.VerifiedBumpCount != 0 || rel.Band != bump.BandNew {
		t.Fatalf("a user with no BUMPs must default to zero/NEW, got %#v", rel)
	}
}
