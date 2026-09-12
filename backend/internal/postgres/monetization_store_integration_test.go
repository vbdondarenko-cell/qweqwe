package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

func rewardedHash(provider, viewID string) []byte {
	sum := sha256.Sum256([]byte(provider + ":" + viewID))
	return sum[:]
}

func purchaseHash(provider, purchaseID string) []byte {
	sum := sha256.Sum256([]byte(provider + ":" + purchaseID))
	return sum[:]
}

// TestMonetizationStoreRewardedQuestGrantsPremiumAfterFiveVerifiedViews
// proves docs/LINKUP_PLUS_MONETIZATION.md §6's rewarded-quest state machine
// end to end against real PostgreSQL: five verified views spaced at least
// 4h apart within the 24h quest window atomically grant 24h of premium on
// the fifth; a replay of that exact same fifth view never double-grants;
// a fresh view submitted during the 7-day claim cooldown is rejected
// without advancing anything; and a view submitted too soon after the
// previous one is rejected too.
func TestMonetizationStoreRewardedQuestGrantsPremiumAfterFiveVerifiedViews(t *testing.T) {
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
	assertMigrationCount(t, ctx, pool, 38)

	accountService, err := account.NewService(NewAccountStore(pool), password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	suffix := fmt.Sprintf("%x", time.Now().UnixNano())
	user := registerIntegrationUser(t, ctx, accountService, "rwq", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{user.User.ID}) })

	store := NewMonetizationStore(pool)
	const (
		minInterval   = 4 * time.Hour
		questWindow   = 24 * time.Hour
		grantDuration = 24 * time.Hour
		claimCooldown = 7 * 24 * time.Hour
	)
	t0 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	// Too-soon rejection: a second view less than 4h after the first must
	// not advance progress.
	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "v1"), t0, minInterval, questWindow, grantDuration, claimCooldown); err != nil {
		t.Fatalf("view 1: %v", err)
	}
	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "v1b"), t0.Add(time.Hour), minInterval, questWindow, grantDuration, claimCooldown); err != monetization.ErrRewardedTooSoon {
		t.Fatalf("expected ErrRewardedTooSoon, got %v", err)
	}

	// Continue the quest correctly-spaced: views 2-5 at t0+4h, +8h, +12h, +16h.
	var lastViewAt time.Time
	for i, hoursOffset := range []int{4, 8, 12, 16} {
		lastViewAt = t0.Add(time.Duration(hoursOffset) * time.Hour)
		viewID := fmt.Sprintf("v%d", i+2)
		if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", viewID), lastViewAt, minInterval, questWindow, grantDuration, claimCooldown); err != nil {
			t.Fatalf("view %d: %v", i+2, err)
		}
	}

	status, err := store.Status(ctx, user.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantUntil := lastViewAt.Add(grantDuration)
	if status.PremiumUntil == nil || !status.PremiumUntil.Equal(wantUntil) {
		t.Fatalf("premiumUntil=%v want=%v", status.PremiumUntil, wantUntil)
	}
	if status.VideosWatchedCount != 0 {
		t.Fatalf("progress should reset after granting, got count=%d", status.VideosWatchedCount)
	}
	if status.LastFreePremiumClaimedAt == nil || !status.LastFreePremiumClaimedAt.Equal(lastViewAt) {
		t.Fatalf("lastFreePremiumClaimedAt=%v want=%v", status.LastFreePremiumClaimedAt, lastViewAt)
	}

	// Replaying the exact same 5th view must not grant a second time.
	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "v5"), lastViewAt, minInterval, questWindow, grantDuration, claimCooldown); err != nil {
		t.Fatalf("replay of view 5: %v", err)
	}
	replayed, err := store.Status(ctx, user.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !replayed.PremiumUntil.Equal(wantUntil) {
		t.Fatalf("replay must not extend/duplicate the grant: got %v want %v", replayed.PremiumUntil, wantUntil)
	}

	// A fresh view during the 7-day cooldown is rejected outright.
	duringCooldown := lastViewAt.Add(time.Hour)
	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "v6-during-cooldown"), duringCooldown, minInterval, questWindow, grantDuration, claimCooldown); err != monetization.ErrRewardedCooldownActive {
		t.Fatalf("expected ErrRewardedCooldownActive, got %v", err)
	}
	afterRejected, err := store.Status(ctx, user.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterRejected.VideosWatchedCount != 0 {
		t.Fatalf("a cooldown-rejected view must not start a new quest, got count=%d", afterRejected.VideosWatchedCount)
	}
}

// TestMonetizationStoreRewardedQuestWindowExpiryRestartsRatherThanErrors
// proves the 24h quest window is measured from the FIRST verified view of
// the quest, not the most recent one, and that a view arriving after that
// window still counts -- it just restarts the quest instead of being lost.
func TestMonetizationStoreRewardedQuestWindowExpiryRestartsRatherThanErrors(t *testing.T) {
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
	user := registerIntegrationUser(t, ctx, accountService, "rwe", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{user.User.ID}) })

	store := NewMonetizationStore(pool)
	const (
		minInterval   = 4 * time.Hour
		questWindow   = 24 * time.Hour
		grantDuration = 24 * time.Hour
		claimCooldown = 7 * 24 * time.Hour
	)
	t0 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)

	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "w1"), t0, minInterval, questWindow, grantDuration, claimCooldown); err != nil {
		t.Fatal(err)
	}
	// Well past the 24h window -- this must restart the quest at count=1,
	// not error and not silently continue toward 5.
	restart := t0.Add(30 * time.Hour)
	if err := store.RecordRewardedView(ctx, user.User.ID, "admob", rewardedHash("admob", "w2"), restart, minInterval, questWindow, grantDuration, claimCooldown); err != nil {
		t.Fatal(err)
	}
	status, err := store.Status(ctx, user.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status.VideosWatchedCount != 1 {
		t.Fatalf("expected the expired-window view to restart the quest at 1, got %d", status.VideosWatchedCount)
	}
	if status.PremiumUntil != nil {
		t.Fatal("no premium should be granted yet")
	}
}

// TestMonetizationStorePurchaseGrantsEntitlementAndQualifiesReferral proves
// docs/LINKUP_PLUS_MONETIZATION.md §7/§8 end to end: an ACTIVE verified
// purchase grants PAID premium; it also qualifies a pending referral (if
// any) and awards the crossed milestone to both inviter and the triggering
// invitee exactly once, even under a replayed verification; and a later
// REVOKED verification of the SAME purchase retracts only that purchase's
// own grant, leaving the referral-milestone grants untouched.
func TestMonetizationStorePurchaseGrantsEntitlementAndQualifiesReferral(t *testing.T) {
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
	inviter := registerIntegrationUser(t, ctx, accountService, "pinvr", suffix)
	invitee := registerIntegrationUser(t, ctx, accountService, "pinve", suffix)
	t.Cleanup(func() { cleanupIntegrationRows(pool, []string{inviter.User.ID, invitee.User.ID}) })

	store := NewMonetizationStore(pool)
	code, err := store.EnsureReferralCode(ctx, inviter.User.ID, "PURCHTEST01")
	if err != nil {
		t.Fatal(err)
	}
	if err := store.BindReferral(ctx, invitee.User.ID, code, 14); err != nil {
		t.Fatal(err)
	}

	milestones := []monetization.ReferralMilestone{
		{QualifiedReferrals: 1, InviterRewardDays: 1, InviteeRewardDays: 1},
	}
	periodStart := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	periodEnd := periodStart.Add(30 * 24 * time.Hour)
	now := periodStart
	hash := purchaseHash("play", "purchase-1")

	if err := store.RecordPurchase(ctx, invitee.User.ID, "play", "linkup_plus_monthly", hash, periodStart, periodEnd, "ACTIVE", now, milestones); err != nil {
		t.Fatal(err)
	}

	inviteeStatus, err := store.Status(ctx, invitee.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	if inviteeStatus.PremiumUntil == nil || !inviteeStatus.PremiumUntil.Equal(periodEnd) {
		t.Fatalf("invitee premiumUntil=%v want=%v (30-day purchase should dominate the 1-day referral bonus)", inviteeStatus.PremiumUntil, periodEnd)
	}

	inviterStatus, err := store.Status(ctx, inviter.User.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantInviterUntil := now.Add(24 * time.Hour)
	if inviterStatus.PremiumUntil == nil || !inviterStatus.PremiumUntil.Equal(wantInviterUntil) {
		t.Fatalf("inviter premiumUntil=%v want=%v", inviterStatus.PremiumUntil, wantInviterUntil)
	}
	if inviterStatus.QualifiedReferrals != 1 {
		t.Fatalf("inviter qualified referrals=%d want=1", inviterStatus.QualifiedReferrals)
	}

	var grantCount, milestoneCount int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM premium_grants WHERE user_id=$1`, invitee.User.ID).Scan(&grantCount); err != nil {
		t.Fatal(err)
	}
	if grantCount != 2 {
		t.Fatalf("invitee should have both a PAID grant and a REFERRAL trigger-bonus grant, got %d rows", grantCount)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monetization_referral_milestone_awards WHERE inviter_id=$1 AND milestone=1`, inviter.User.ID).Scan(&milestoneCount); err != nil {
		t.Fatal(err)
	}
	if milestoneCount != 1 {
		t.Fatalf("milestone award rows=%d want=1", milestoneCount)
	}

	// Replaying the exact same ACTIVE verification must not double-award.
	if err := store.RecordPurchase(ctx, invitee.User.ID, "play", "linkup_plus_monthly", hash, periodStart, periodEnd, "ACTIVE", now, milestones); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM monetization_referral_milestone_awards WHERE inviter_id=$1 AND milestone=1`, inviter.User.ID).Scan(&milestoneCount); err != nil {
		t.Fatal(err)
	}
	if milestoneCount != 1 {
		t.Fatalf("replay must not duplicate the milestone award, got %d rows", milestoneCount)
	}

	// A later REVOKED verification retracts only the PAID grant.
	if err := store.RecordPurchase(ctx, invitee.User.ID, "play", "linkup_plus_monthly", hash, periodStart, periodEnd, "REVOKED", now.Add(time.Hour), milestones); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM premium_grants WHERE user_id=$1`, invitee.User.ID).Scan(&grantCount); err != nil {
		t.Fatal(err)
	}
	if grantCount != 1 {
		t.Fatalf("revoke should retract only the PAID grant, leaving the REFERRAL bonus in place: got %d rows", grantCount)
	}
	var remainingSource string
	if err := pool.QueryRow(ctx, `SELECT source FROM premium_grants WHERE user_id=$1`, invitee.User.ID).Scan(&remainingSource); err != nil {
		t.Fatal(err)
	}
	if remainingSource != "REFERRAL" {
		t.Fatalf("remaining grant source=%q want=REFERRAL", remainingSource)
	}
}
