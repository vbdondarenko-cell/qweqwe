package monetization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

type Service struct {
	store        Store
	now          func() time.Time
	rewarded     RewardedVerifier
	purchases    PurchaseVerifier
	capabilities Capabilities
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

// ConfigureRewarded plugs in a real ad-network SSV adapter, enabling
// SubmitRewardedView and flipping Capabilities.RewardedVerification on.
// Mirrors account.Service.ConfigureRecovery's optional-dependency shape —
// until this is called, SubmitRewardedView fails closed.
func (s *Service) ConfigureRewarded(v RewardedVerifier) {
	s.rewarded = v
	s.capabilities.RewardedVerification = v != nil
}

// ConfigurePurchases plugs in a real store (Google Play/App Store) purchase
// verifier, enabling VerifyPurchase. Referral qualification depends on a
// real paid-purchase signal, so it is tied to the same capability rather
// than needing its own separate configuration call.
func (s *Service) ConfigurePurchases(v PurchaseVerifier) {
	s.purchases = v
	s.capabilities.PaidVerification = v != nil
	s.capabilities.ReferralQualification = v != nil
}

func (s *Service) Catalog() Catalog {
	return Catalog{
		Currency:             "UAH",
		Plans:                plans(),
		RecommendedPlanID:    "THREE_MONTH",
		ReferralDeadlineDays: ReferralDeadlineDays,
		ReferralMilestones:   referralMilestones(),
		Benefits:             benefits(),
		Rewarded: RewardedPolicy{
			VideoIntervalSeconds:   int64(rewardedMinInterval / time.Second),
			VideosRequired:         RewardedVideosRequired,
			NominalCompletionHours: RewardedNominalHours,
			QuestWindowSeconds:     int64(rewardedQuestWindow / time.Second),
			RewardSeconds:          int64(rewardedGrantDuration / time.Second),
			ClaimCooldownSeconds:   int64(rewardedClaimCooldown / time.Second),
		},
	}
}

// plans returns the four launch-grid durations in the paywall's
// recommended display order (docs/LINKUP_PLUS_MONETIZATION.md §2), with
// each plan's own price/effective-monthly/savings computed from the exact
// launch prices in that section's table — never re-derived from a
// different formula that could silently drift from the documented numbers.
func plans() []Plan {
	monthly := MonthlyPriceUAHMinor
	return []Plan{
		{
			ID: "THREE_MONTH", Label: "THREE_MONTH", PriceUAHMinor: ThreeMonthPriceUAHMinor,
			EffectiveMonthlyUAHMinor: 8333,
			SavingsVsMonthlyUAHMinor: 3*monthly - ThreeMonthPriceUAHMinor,
			SavingsVsMonthlyPercent:  16.7,
			Recommended:              true,
		},
		{
			ID: "MONTHLY", Label: "MONTH", PriceUAHMinor: monthly,
			EffectiveMonthlyUAHMinor: monthly,
		},
		{
			ID: "ANNUAL", Label: "YEAR", PriceUAHMinor: AnnualPriceUAHMinor,
			EffectiveMonthlyUAHMinor: 6667,
			SavingsVsMonthlyUAHMinor: 12*monthly - AnnualPriceUAHMinor,
			SavingsVsMonthlyPercent:  33.3,
		},
		{
			ID: "WEEKLY", Label: "WEEK", PriceUAHMinor: WeeklyPriceUAHMinor,
		},
	}
}

// benefits reports the honest, currently-fail-closed availability of every
// still-unbuilt LinkUp+ feature area (README §6.21-§6.26). None of Travel
// Pro/Host Power Tools/Advanced Discovery/Privacy&QoL/Identity-Analytics/
// Forgiveness has any server-side implementation yet — confirmed by
// grepping the whole backend for their canonical terms (Mega-Slot, Stealth
// Slot, Ghost Mode, Guardian Auto-Ping, Hex-Aura, No-Strike) before writing
// this, not assumed. Every entry is therefore Available:false; this exists
// so the paywall/UX can honestly show "coming soon", never "included and
// active" (docs/LINKUP_PLUS_MONETIZATION.md §9's own explicit ban on that).
func benefits() []Benefit {
	return []Benefit{
		{ID: BenefitTravelPro},
		{ID: BenefitHostPowerTools},
		{ID: BenefitAdvancedDiscovery},
		{ID: BenefitPrivacyQoL},
		{ID: BenefitIdentityAnalytics},
		{ID: BenefitForgiveness},
	}
}

func (s *Service) Snapshot(ctx context.Context, userID string) (Snapshot, error) {
	if s == nil || s.store == nil || userID == "" {
		return Snapshot{}, ErrUnavailable
	}
	code, err := s.store.EnsureReferralCode(ctx, userID, referralCodeForUser(userID))
	if err != nil {
		return Snapshot{}, err
	}
	raw, err := s.store.Status(ctx, userID)
	if err != nil {
		return Snapshot{}, err
	}
	now := s.now().UTC()
	status := Status{
		PremiumUntil:  raw.PremiumUntil,
		PremiumActive: raw.PremiumUntil != nil && raw.PremiumUntil.After(now),
		Rewarded: RewardedStatus{
			VideosWatchedCount:       raw.VideosWatchedCount,
			LastFreePremiumClaimedAt: raw.LastFreePremiumClaimedAt,
		},
		Referral: ReferralStatus{
			ReferralCode:       code,
			BoundReferralCode:  raw.BoundReferralCode,
			QualifyingDeadline: raw.ReferralQualifyingDeadline,
			QualifiedReferrals: raw.QualifiedReferrals,
			NextMilestone:      nextMilestone(raw.QualifiedReferrals),
		},
	}
	if raw.LastVideoWatchedAt != nil {
		next := raw.LastVideoWatchedAt.Add(rewardedMinInterval)
		status.Rewarded.NextVideoAt = &next
	}
	if raw.LastFreePremiumClaimedAt != nil {
		next := raw.LastFreePremiumClaimedAt.Add(rewardedClaimCooldown)
		status.Rewarded.NextFreePremiumClaimAt = &next
	}
	return Snapshot{Catalog: s.Catalog(), Status: status, Capabilities: s.capabilities}, nil
}

func (s *Service) BindReferral(ctx context.Context, userID, rawCode string) error {
	if s == nil || s.store == nil || userID == "" {
		return ErrUnavailable
	}
	code := strings.ToUpper(strings.TrimSpace(rawCode))
	if !validReferralCode(code) {
		return ErrInvalidReferralCode
	}
	return s.store.BindReferral(ctx, userID, code, ReferralDeadlineDays)
}

// SubmitRewardedView verifies one rewarded-ad-watch event with the
// configured ad network before it ever reaches the Store — a client can
// only trigger verification, never assert the outcome
// (docs/LINKUP_PLUS_MONETIZATION.md §6: no client-authoritative
// watched=true). Only the network's own verified view identity (hashed) is
// persisted; the raw receipt/token never is.
func (s *Service) SubmitRewardedView(ctx context.Context, userID, rawReceipt string) (Snapshot, error) {
	if s == nil || s.store == nil || userID == "" {
		return Snapshot{}, ErrUnavailable
	}
	if s.rewarded == nil {
		return Snapshot{}, ErrCapabilityDisabled
	}
	provider, viewID, err := s.rewarded.VerifyRewardedView(ctx, userID, rawReceipt)
	if err != nil || strings.TrimSpace(provider) == "" || strings.TrimSpace(viewID) == "" {
		return Snapshot{}, ErrRewardedViewRejected
	}
	hash := sha256.Sum256([]byte(provider + ":" + viewID))
	now := s.now().UTC()
	if err := s.store.RecordRewardedView(ctx, userID, provider, hash[:], now,
		rewardedMinInterval, rewardedQuestWindow, rewardedGrantDuration, rewardedClaimCooldown); err != nil {
		return Snapshot{}, err
	}
	return s.Snapshot(ctx, userID)
}

// VerifyPurchase checks a subscription purchase with the configured store
// verifier before recording anything — same client-cannot-assert-outcome
// boundary as SubmitRewardedView.
func (s *Service) VerifyPurchase(ctx context.Context, userID, rawPurchaseToken string) (Snapshot, error) {
	if s == nil || s.store == nil || userID == "" {
		return Snapshot{}, ErrUnavailable
	}
	if s.purchases == nil {
		return Snapshot{}, ErrCapabilityDisabled
	}
	verified, err := s.purchases.VerifyPurchase(ctx, userID, rawPurchaseToken)
	if err != nil {
		return Snapshot{}, ErrPurchaseRejected
	}
	if !validSubscriptionState(verified.State) || strings.TrimSpace(verified.Provider) == "" ||
		strings.TrimSpace(verified.PurchaseID) == "" || strings.TrimSpace(verified.ProductID) == "" ||
		!verified.PeriodEnd.After(verified.PeriodStart) {
		return Snapshot{}, ErrInvalidPurchase
	}
	hash := sha256.Sum256([]byte(verified.Provider + ":" + verified.PurchaseID))
	now := s.now().UTC()
	if err := s.store.RecordPurchase(ctx, userID, verified.Provider, verified.ProductID, hash[:],
		verified.PeriodStart.UTC(), verified.PeriodEnd.UTC(), verified.State, now,
		referralMilestones()); err != nil {
		return Snapshot{}, err
	}
	return s.Snapshot(ctx, userID)
}

func validSubscriptionState(state string) bool {
	switch state {
	case "ACTIVE", "GRACE", "BILLING_RETRY", "EXPIRED", "REVOKED", "REFUNDED":
		return true
	default:
		return false
	}
}

func referralCodeForUser(userID string) string {
	sum := sha256.Sum256([]byte(userID))
	return strings.ToUpper(hex.EncodeToString(sum[:8]))
}

func validReferralCode(code string) bool {
	if len(code) < 6 || len(code) > 20 {
		return false
	}
	for _, r := range code {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func referralMilestones() []ReferralMilestone {
	return []ReferralMilestone{
		{QualifiedReferrals: 1, InviterRewardDays: 1, InviteeRewardDays: 1},
		{QualifiedReferrals: 3, InviterRewardDays: 7, InviteeRewardDays: 3},
		{QualifiedReferrals: 5, InviterRewardDays: 30, InviteeRewardDays: 7},
		{QualifiedReferrals: 10, InviterRewardDays: 90, InviteeRewardDays: 7, Badge: true},
	}
}

func nextMilestone(count int) *ReferralMilestone {
	for _, item := range referralMilestones() {
		if count < item.QualifiedReferrals {
			copy := item
			return &copy
		}
	}
	return nil
}
