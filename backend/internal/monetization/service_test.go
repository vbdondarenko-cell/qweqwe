package monetization

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeStore struct {
	status               StoreStatus
	boundCode            string
	bindErr              error
	rewardedErr          error
	lastRewardedProvider string
	rewardedCalls        int
	purchaseErr          error
	lastPurchaseState    string
	purchaseCalls        int
}

func (f *fakeStore) Status(context.Context, string) (StoreStatus, error) { return f.status, nil }
func (f *fakeStore) EnsureReferralCode(_ context.Context, _ string, code string) (string, error) {
	return code, nil
}
func (f *fakeStore) BindReferral(_ context.Context, _ string, code string, _ int) error {
	f.boundCode = code
	return f.bindErr
}
func (f *fakeStore) RecordRewardedView(_ context.Context, _, provider string, _ []byte, _ time.Time, _, _, _, _ time.Duration) error {
	f.rewardedCalls++
	f.lastRewardedProvider = provider
	return f.rewardedErr
}
func (f *fakeStore) RecordPurchase(_ context.Context, _, _, _ string, _ []byte, _, _ time.Time, state string, _ time.Time, _ []ReferralMilestone) error {
	f.purchaseCalls++
	f.lastPurchaseState = state
	return f.purchaseErr
}

type fakeRewardedVerifier struct {
	provider string
	viewID   string
	err      error
}

func (f *fakeRewardedVerifier) VerifyRewardedView(context.Context, string, string) (string, string, error) {
	return f.provider, f.viewID, f.err
}

type fakePurchaseVerifier struct {
	result VerifiedPurchase
	err    error
}

func (f *fakePurchaseVerifier) VerifyPurchase(context.Context, string, string) (VerifiedPurchase, error) {
	return f.result, f.err
}

func TestCatalogMatchesProductContract(t *testing.T) {
	s := NewService(&fakeStore{})
	catalog := s.Catalog()
	if len(catalog.Plans) != 4 {
		t.Fatalf("plans=%d", len(catalog.Plans))
	}
	byID := map[string]Plan{}
	for _, p := range catalog.Plans {
		byID[p.ID] = p
	}
	weekly, monthly, three, annual := byID["WEEKLY"], byID["MONTHLY"], byID["THREE_MONTH"], byID["ANNUAL"]

	if weekly.PriceUAHMinor != 3999 || weekly.EffectiveMonthlyUAHMinor != 0 {
		t.Fatalf("weekly=%+v", weekly)
	}
	if monthly.PriceUAHMinor != 9999 || monthly.EffectiveMonthlyUAHMinor != 9999 {
		t.Fatalf("monthly=%+v", monthly)
	}
	if three.PriceUAHMinor != 24999 || three.EffectiveMonthlyUAHMinor != 8333 ||
		three.SavingsVsMonthlyUAHMinor != 4998 || three.SavingsVsMonthlyPercent != 16.7 || !three.Recommended {
		t.Fatalf("three_month=%+v", three)
	}
	if annual.PriceUAHMinor != 79999 || annual.EffectiveMonthlyUAHMinor != 6667 ||
		annual.SavingsVsMonthlyUAHMinor != 39989 || annual.SavingsVsMonthlyPercent != 33.3 {
		t.Fatalf("annual=%+v", annual)
	}
	if catalog.RecommendedPlanID != "THREE_MONTH" {
		t.Fatalf("recommended plan id=%q", catalog.RecommendedPlanID)
	}
	if catalog.Rewarded.VideoIntervalSeconds != 4*60*60 || catalog.Rewarded.VideosRequired != 5 ||
		catalog.Rewarded.NominalCompletionHours != 20 || catalog.Rewarded.QuestWindowSeconds != 24*60*60 {
		t.Fatalf("rewarded=%+v", catalog.Rewarded)
	}
	if catalog.Rewarded.RewardSeconds != 24*60*60 || catalog.Rewarded.ClaimCooldownSeconds != 7*24*60*60 {
		t.Fatalf("rewarded durations=%+v", catalog.Rewarded)
	}
	if catalog.ReferralDeadlineDays != 14 || len(catalog.ReferralMilestones) != 4 {
		t.Fatalf("referrals=%+v", catalog)
	}
	if len(catalog.Benefits) != 6 {
		t.Fatalf("benefits=%+v", catalog.Benefits)
	}
	for _, b := range catalog.Benefits {
		if b.Available {
			t.Fatalf("benefit %q must be fail-closed until its feature actually ships: %+v", b.ID, b)
		}
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
		PremiumUntil:               &premium,
		VideosWatchedCount:         3,
		LastVideoWatchedAt:         &lastVideo,
		LastFreePremiumClaimedAt:   &lastClaim,
		QualifiedReferrals:         4,
		BoundReferralCode:          &bound,
		ReferralQualifyingDeadline: &deadline,
	}})
	s.now = func() time.Time { return now }

	snapshot, err := s.Snapshot(context.Background(), "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.Status.PremiumActive {
		t.Fatal("premium should be active")
	}
	if snapshot.Status.Rewarded.NextVideoAt == nil || !snapshot.Status.Rewarded.NextVideoAt.Equal(now.Add(3*time.Hour)) {
		t.Fatalf("nextVideoAt=%v", snapshot.Status.Rewarded.NextVideoAt)
	}
	if snapshot.Status.Rewarded.NextFreePremiumClaimAt == nil || !snapshot.Status.Rewarded.NextFreePremiumClaimAt.Equal(lastClaim.Add(7*24*time.Hour)) {
		t.Fatalf("nextClaim=%v", snapshot.Status.Rewarded.NextFreePremiumClaimAt)
	}
	if snapshot.Status.Referral.ReferralCode == "" {
		t.Fatal("referral code missing")
	}
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
	if err := s.BindReferral(context.Background(), "user-1", " ab12cd "); err != nil {
		t.Fatal(err)
	}
	if store.boundCode != "AB12CD" {
		t.Fatalf("bound code=%q", store.boundCode)
	}

	for _, invalid := range []string{"", "short", "A B C 1", "abcdef-123"} {
		if err := s.BindReferral(context.Background(), "user-1", invalid); err != ErrInvalidReferralCode {
			t.Fatalf("code=%q err=%v", invalid, err)
		}
	}
}

func TestReferralCodeIsStableUppercaseAndBounded(t *testing.T) {
	first := referralCodeForUser("user-1")
	second := referralCodeForUser("user-1")
	if first != second {
		t.Fatalf("unstable code %q %q", first, second)
	}
	if len(first) != 16 || !validReferralCode(first) {
		t.Fatalf("invalid generated code %q", first)
	}
}

func TestSubmitRewardedViewFailsClosedWithoutVerifier(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	if _, err := s.SubmitRewardedView(context.Background(), "user-1", "raw-receipt"); !errors.Is(err, ErrCapabilityDisabled) {
		t.Fatalf("err=%v", err)
	}
	if store.rewardedCalls != 0 {
		t.Fatal("store must not be touched when the capability is disabled")
	}
}

func TestSubmitRewardedViewRejectsUnverifiedReceipt(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	s.ConfigureRewarded(&fakeRewardedVerifier{err: errors.New("provider rejected")})
	if _, err := s.SubmitRewardedView(context.Background(), "user-1", "raw-receipt"); !errors.Is(err, ErrRewardedViewRejected) {
		t.Fatalf("err=%v", err)
	}
	if store.rewardedCalls != 0 {
		t.Fatal("an unverified view must never reach the store")
	}
}

func TestSubmitRewardedViewRecordsVerifiedProviderAndHashesReceipt(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	s.ConfigureRewarded(&fakeRewardedVerifier{provider: "admob", viewID: "view-123"})
	if _, err := s.SubmitRewardedView(context.Background(), "user-1", "raw-receipt-should-never-be-stored"); err != nil {
		t.Fatal(err)
	}
	if store.rewardedCalls != 1 || store.lastRewardedProvider != "admob" {
		t.Fatalf("calls=%d provider=%q", store.rewardedCalls, store.lastRewardedProvider)
	}
}

func TestSubmitRewardedViewPropagatesStoreErrors(t *testing.T) {
	store := &fakeStore{rewardedErr: ErrRewardedTooSoon}
	s := NewService(store)
	s.ConfigureRewarded(&fakeRewardedVerifier{provider: "admob", viewID: "view-123"})
	if _, err := s.SubmitRewardedView(context.Background(), "user-1", "raw"); !errors.Is(err, ErrRewardedTooSoon) {
		t.Fatalf("err=%v", err)
	}
}

func TestVerifyPurchaseFailsClosedWithoutVerifier(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	if _, err := s.VerifyPurchase(context.Background(), "user-1", "raw-token"); !errors.Is(err, ErrCapabilityDisabled) {
		t.Fatalf("err=%v", err)
	}
	if store.purchaseCalls != 0 {
		t.Fatal("store must not be touched when the capability is disabled")
	}
}

func TestVerifyPurchaseRejectsMalformedVerifiedResult(t *testing.T) {
	now := time.Now()
	cases := []VerifiedPurchase{
		{Provider: "", ProductID: "p", PurchaseID: "id", State: "ACTIVE", PeriodStart: now, PeriodEnd: now.Add(time.Hour)},
		{Provider: "play", ProductID: "p", PurchaseID: "", State: "ACTIVE", PeriodStart: now, PeriodEnd: now.Add(time.Hour)},
		{Provider: "play", ProductID: "p", PurchaseID: "id", State: "NOT_A_REAL_STATE", PeriodStart: now, PeriodEnd: now.Add(time.Hour)},
		{Provider: "play", ProductID: "p", PurchaseID: "id", State: "ACTIVE", PeriodStart: now, PeriodEnd: now},
	}
	for _, c := range cases {
		store := &fakeStore{}
		s := NewService(store)
		s.ConfigurePurchases(&fakePurchaseVerifier{result: c})
		if _, err := s.VerifyPurchase(context.Background(), "user-1", "raw-token"); !errors.Is(err, ErrInvalidPurchase) {
			t.Fatalf("case=%+v err=%v", c, err)
		}
		if store.purchaseCalls != 0 {
			t.Fatalf("malformed result must never reach the store: %+v", c)
		}
	}
}

func TestVerifyPurchaseRecordsNormalizedStateAndEnablesReferralQualification(t *testing.T) {
	store := &fakeStore{}
	s := NewService(store)
	s.ConfigurePurchases(&fakePurchaseVerifier{result: VerifiedPurchase{
		Provider: "play", ProductID: "linkup_plus_monthly", PurchaseID: "purchase-1",
		State: "ACTIVE", PeriodStart: time.Now(), PeriodEnd: time.Now().Add(30 * 24 * time.Hour),
	}})
	if !s.capabilities.ReferralQualification {
		t.Fatal("configuring a purchase verifier must enable referral qualification")
	}
	if _, err := s.VerifyPurchase(context.Background(), "user-1", "raw-token"); err != nil {
		t.Fatal(err)
	}
	if store.purchaseCalls != 1 || store.lastPurchaseState != "ACTIVE" {
		t.Fatalf("calls=%d state=%q", store.purchaseCalls, store.lastPurchaseState)
	}
}
