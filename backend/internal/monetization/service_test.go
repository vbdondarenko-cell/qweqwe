package monetization

import (
	"context"
	"testing"
	"time"
)

type fakeStore struct {
	status    StoreStatus
	boundCode string
	bindErr   error
}

func (f *fakeStore) Status(context.Context, string) (StoreStatus, error) { return f.status, nil }
func (f *fakeStore) EnsureReferralCode(_ context.Context, _ string, code string) (string, error) { return code, nil }
func (f *fakeStore) BindReferral(_ context.Context, _ string, code string, _ int) error {
	f.boundCode = code
	return f.bindErr
}

func TestCatalogMatchesProductContract(t *testing.T) {
	s := NewService(&fakeStore{})
	catalog := s.Catalog()
	if len(catalog.Plans) != 2 { t.Fatalf("plans=%d", len(catalog.Plans)) }
	if catalog.Plans[0].PriceUAHMinor != 14999 { t.Fatalf("monthly=%d", catalog.Plans[0].PriceUAHMinor) }
	if catalog.Plans[1].EffectiveMonthlyUAHMinor != 9999 || catalog.Plans[1].Total12MonthsUAHMinor != 119988 {
		t.Fatalf("annual=%+v", catalog.Plans[1])
	}
	if catalog.AnnualSavingsUAHMinor != 60000 || catalog.AnnualSavingsPercent != 33.3 {
		t.Fatalf("savings=%d %.1f", catalog.AnnualSavingsUAHMinor, catalog.AnnualSavingsPercent)
	}
	if catalog.Rewarded.VideoIntervalSeconds != 4*60*60 || catalog.Rewarded.VideosRequired != 5 || catalog.Rewarded.NominalCompletionHours != 20 {
		t.Fatalf("rewarded=%+v", catalog.Rewarded)
	}
	if catalog.Rewarded.RewardSeconds != 24*60*60 || catalog.Rewarded.ClaimCooldownSeconds != 7*24*60*60 {
		t.Fatalf("rewarded durations=%+v", catalog.Rewarded)
	}
	if catalog.ReferralDeadlineDays != 14 || len(catalog.ReferralMilestones) != 4 {
		t.Fatalf("referrals=%+v", catalog)
	}
}

func TestSnapshotDerivesPremiumAndNextWindows(t *testing.T) {
	now := time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC)
	premium := now.Add(24 * time.Hour)
	lastVideo := now.Add(-time.Hour)
	lastClaim := now.Add(-48 * time.Hour)
	bound := "ABC123"
	deadline := now.Add(8 * 24 * time.Hour)
	s := NewService(&fakeStore{status: StoreStatus{
		PremiumUntil: &premium,
		VideosWatchedCount: 3,
		LastVideoWatchedAt: &lastVideo,
		LastFreePremiumClaimedAt: &lastClaim,
		QualifiedReferrals: 4,
		BoundReferralCode: &bound,
		ReferralQualifyingDeadline: &deadline,
	}})
	s.now = func() time.Time { return now }

	snapshot, err := s.Snapshot(context.Background(), "user-1")
	if err != nil { t.Fatal(err) }
	if !snapshot.Status.PremiumActive { t.Fatal("premium should be active") }
	if snapshot.Status.Rewarded.NextVideoAt == nil || !snapshot.Status.Rewarded.NextVideoAt.Equal(now.Add(3*time.Hour)) {
		t.Fatalf("nextVideoAt=%v", snapshot.Status.Rewarded.NextVideoAt)
	}
	if snapshot.Status.Rewarded.NextFreePremiumClaimAt == nil || !snapshot.Status.Rewarded.NextFreePremiumClaimAt.Equal(lastClaim.Add(7*24*time.Hour)) {
		t.Fatalf("nextClaim=%v", snapshot.Status.Rewarded.NextFreePremiumClaimAt)
	}
	if snapshot.Status.Referral.ReferralCode == "" { t.Fatal("referral code missing") }
	if snapshot.Status.Referral.BoundReferralCode == nil || *snapshot.Status.Referral.BoundReferralCode != bound {
		t.Fatalf("bound referral=%v", snapshot.Status.Referral.BoundReferralCode)
	}
	if snapshot.Status.Referral.NextMilestone == nil || snapshot.Status.Referral.NextMilestone.QualifiedReferrals != 5 {
		t.Fatalf("nextMilestone=%+v", snapshot.Status.Referral.NextMilestone)
	}
	if snapshot.Capabilities.PaidVerification || snapshot.Capabilities.RewardedVerification || snapshot.Capabilities.ReferralQualification {
		t.Fatal("provider-backed capabilities must fail closed until configured")
	}
}

func TestBindReferralNormalizesAndRejectsInvalidCode(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	if err := s.BindReferral(context.Background(), "user-1", " ab12cd "); err != nil { t.Fatal(err) }
	if store.boundCode != "AB12CD" { t.Fatalf("bound code=%q", store.boundCode) }

	for _, invalid := range []string{"", "short", "A B C 1", "abcdef-123"} {
		if err := s.BindReferral(context.Background(), "user-1", invalid); err != ErrInvalidReferralCode {
			t.Fatalf("code=%q err=%v", invalid, err)
		}
	}
}

func TestReferralCodeIsStableUppercaseAndBounded(t *testing.T) {
	first := referralCodeForUser("user-1")
	second := referralCodeForUser("user-1")
	if first != second { t.Fatalf("unstable code %q %q", first, second) }
	if len(first) != 16 || !validReferralCode(first) { t.Fatalf("invalid generated code %q", first) }
}
