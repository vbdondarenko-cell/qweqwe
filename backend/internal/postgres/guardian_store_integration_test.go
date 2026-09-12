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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/guardian"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// TestV11GuardianLinkLifecycleAgainstRealPostgres proves README §6.17's
// Ghost Guardian STATUS_ONLY lifecycle end to end: a stranger cannot
// create a link for someone else's Slot; the host and an accepted member
// both can; access with a valid token reveals only the sharer's display
// name and a coarse ACTIVE/ENDED status (never the Slot's title, place, or
// roster); revoke, expiry, and a token that never existed all converge to
// the exact same ErrLinkUnavailable outcome ("indistinguishable/burned
// access").
func TestV11GuardianLinkLifecycleAgainstRealPostgres(t *testing.T) {
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
	assertMigrationCount(t, ctx, pool, 39)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	host := registerIntegrationUser(t, ctx, accountService, "gdh", suffix)
	member := registerIntegrationUser(t, ctx, accountService, "gdm", suffix)
	stranger := registerIntegrationUser(t, ctx, accountService, "gds", suffix)
	userIDs := []string{host.User.ID, member.User.ID, stranger.User.ID}
	t.Cleanup(func() { cleanupIntegrationRows(pool, userIDs) })

	baseStore, err := NewSlotStore(pool, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotService, err := slot.NewService(baseStore)
	if err != nil {
		t.Fatal(err)
	}
	guardianService, err := guardian.NewService(NewGuardianStore(pool))
	if err != nil {
		t.Fatal(err)
	}

	created, err := slotService.Create(ctx, host.User.ID, slot.CreateInput{
		Title: "Guardian Test Slot", Activity: "run", PlaceText: "Zone", Capacity: 3,
	}, "v11-guardian-create-0001")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := guardianService.CreateLink(ctx, stranger.User.ID, created.ID, guardian.ModeStatusOnly, time.Hour); !errors.Is(err, guardian.ErrForbidden) {
		t.Fatalf("stranger must not be able to create a guardian link, got %v", err)
	}

	if _, err := slotService.Request(ctx, member.User.ID, created.ID, "v11-guardian-request-0001"); err != nil {
		t.Fatal(err)
	}
	if _, err := slotService.Approve(ctx, host.User.ID, created.ID, member.User.ID, "v11-guardian-approve-0001"); err != nil {
		t.Fatal(err)
	}

	memberLink, err := guardianService.CreateLink(ctx, member.User.ID, created.ID, guardian.ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatalf("accepted member must be able to create a guardian link: %v", err)
	}

	status, err := guardianService.Access(ctx, memberLink.Token)
	if err != nil {
		t.Fatal(err)
	}
	if status.SlotStatus != guardian.SlotStatusActive {
		t.Fatalf("expected ACTIVE status for a FILLING slot, got %#v", status)
	}
	if status.SharerDisplayName == "" {
		t.Fatal("expected the sharer's display name")
	}

	// A never-existed token, a revoked token, and (checked separately
	// below) an expired token must all converge to the same
	// ErrLinkUnavailable.
	if _, err := guardianService.Access(ctx, "not-a-real-token-at-all"); !errors.Is(err, guardian.ErrLinkUnavailable) {
		t.Fatalf("expected ErrLinkUnavailable for a nonexistent token, got %v", err)
	}

	hostLink, err := guardianService.CreateLink(ctx, host.User.ID, created.ID, guardian.ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if err := guardianService.Revoke(ctx, host.User.ID, hostLink.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := guardianService.Access(ctx, hostLink.Token); !errors.Is(err, guardian.ErrLinkUnavailable) {
		t.Fatalf("expected ErrLinkUnavailable for a revoked token, got %v", err)
	}
	// A non-creator cannot revoke someone else's link, and a second revoke
	// of an already-revoked link is rejected the same way (no
	// double-revoke, no leaking "already revoked" vs "not yours").
	if err := guardianService.Revoke(ctx, member.User.ID, memberLink.ID); err != nil {
		t.Fatal(err)
	}
	if err := guardianService.Revoke(ctx, member.User.ID, memberLink.ID); !errors.Is(err, guardian.ErrForbidden) {
		t.Fatalf("expected ErrForbidden re-revoking an already-revoked link, got %v", err)
	}
	if err := guardianService.Revoke(ctx, stranger.User.ID, hostLink.ID); !errors.Is(err, guardian.ErrForbidden) {
		t.Fatalf("expected ErrForbidden for a non-creator revoking someone else's link, got %v", err)
	}

	// An expired link (its TTL has already elapsed by the time it's
	// checked) converges to the same ErrLinkUnavailable too. Backdate both
	// created_at and expires_at together so the row still satisfies its
	// own expires_at>created_at CHECK constraint.
	expiringLink, err := guardianService.CreateLink(ctx, host.User.ID, created.ID, guardian.ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `UPDATE guardian_links SET created_at=now() - interval '2 hours', expires_at=now() - interval '1 hour' WHERE id=$1`, expiringLink.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := guardianService.Access(ctx, expiringLink.Token); !errors.Is(err, guardian.ErrLinkUnavailable) {
		t.Fatalf("expected ErrLinkUnavailable for an expired token, got %v", err)
	}

	current, err := slotService.Get(ctx, host.User.ID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	completed, err := slotService.Cancel(ctx, host.User.ID, created.ID, current.Version, "v11-guardian-cancel-0001")
	if err != nil {
		t.Fatal(err)
	}
	if completed.State != slot.StateCancelled {
		t.Fatalf("expected CANCELLED, got %#v", completed)
	}
	// A still-valid (non-expired, non-revoked) link on a now-cancelled
	// Slot legitimately reports ENDED -- that's real status content, not
	// an error.
	endedLink, err := guardianService.CreateLink(ctx, host.User.ID, created.ID, guardian.ModeStatusOnly, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	endedStatus, err := guardianService.Access(ctx, endedLink.Token)
	if err != nil {
		t.Fatal(err)
	}
	if endedStatus.SlotStatus != guardian.SlotStatusEnded {
		t.Fatalf("expected ENDED status for a CANCELLED slot, got %#v", endedStatus)
	}
}
