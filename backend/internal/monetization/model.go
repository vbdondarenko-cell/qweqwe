package monetization

import (
	"context"
	"errors"
	"time"
)

// Pricing per docs/LINKUP_PLUS_MONETIZATION.md §2 — a launch pricing grid,
// explicitly not a proven optimum (that document's own caveat). All amounts
// are minor UAH units (kopecks), matching every other money field in this
// codebase.
const (
	WeeklyPriceUAHMinor     = 3999
	MonthlyPriceUAHMinor    = 9999
	ThreeMonthPriceUAHMinor = 24999
	AnnualPriceUAHMinor     = 79999

	RewardedVideosRequired = 5
	// RewardedNominalHours is a display figure only (5 videos at the
	// minimum interval); rewardedQuestWindow below is the actually enforced
	// clock and is intentionally a separate constant — see its own comment.
	RewardedNominalHours = 20
	ReferralDeadlineDays = 14

	// rewardedMinInterval/rewardedQuestWindow/rewardedGrantDuration/
	// rewardedClaimCooldown implement docs/LINKUP_PLUS_MONETIZATION.md §6
	// exactly: minimum 4h between verified steps, a 24h window measured
	// from the quest's first verified view (not the most recent one), a
	// 24h Plus grant on the 5th verified view, and a 7-day cooldown after
	// that grant before a new quest may start.
	rewardedMinInterval   = 4 * time.Hour
	rewardedQuestWindow   = 24 * time.Hour
	rewardedGrantDuration = 24 * time.Hour
	rewardedClaimCooldown = 7 * 24 * time.Hour
)

// Benefit IDs for the entitlement-benefit/feature-availability catalog
// (docs/LINKUP_PLUS_MONETIZATION.md §3, README §6.21-§6.26). Each maps to
// one README subsection covering an entire not-yet-built feature area.
const (
	BenefitTravelPro         = "TRAVEL_PRO"
	BenefitHostPowerTools    = "HOST_POWER_TOOLS"
	BenefitAdvancedDiscovery = "ADVANCED_DISCOVERY"
	BenefitPrivacyQoL        = "PRIVACY_QOL"
	BenefitIdentityAnalytics = "IDENTITY_ANALYTICS"
	BenefitForgiveness       = "FORGIVENESS"
)

var (
	ErrUnavailable          = errors.New("monetization unavailable")
	ErrInvalidReferralCode  = errors.New("invalid referral code")
	ErrReferralAlreadyBound = errors.New("referral already bound")
	ErrSelfReferral         = errors.New("self referral is not allowed")
	ErrReferralExpired      = errors.New("referral deadline expired")
	// ErrCapabilityDisabled is returned by any provider-backed action
	// (rewarded view, purchase verification) when the corresponding
	// verifier was never configured — the fail-closed boundary
	// README §6.20/§6.27 requires ("client isPlus=true ... is never
	// authority", "provider outage/no-fill не створює fake completion").
	ErrCapabilityDisabled     = errors.New("monetization capability is not configured")
	ErrRewardedViewRejected   = errors.New("rewarded ad view could not be verified")
	ErrRewardedTooSoon        = errors.New("rewarded view submitted before the minimum interval elapsed")
	ErrRewardedCooldownActive = errors.New("rewarded free-premium claim cooldown is still active")
	ErrPurchaseRejected       = errors.New("purchase could not be verified")
	ErrInvalidPurchase        = errors.New("verified purchase response is malformed")
)

// Plan is one selectable LinkUp+ billing duration. Every plan opens the
// identical entitlement (docs/LINKUP_PLUS_MONETIZATION.md §1) — only price
// and duration differ, so Plan carries no per-duration feature list.
type Plan struct {
	ID    string `json:"id"`            // WEEKLY | MONTHLY | THREE_MONTH | ANNUAL
	Label string `json:"billingPeriod"` // WEEK | MONTH | THREE_MONTH | YEAR
	// PriceUAHMinor is the full amount charged for this plan's own period.
	// The real checkout price is always store-returned and localized
	// (§2's own "реальна локальна store price є authority" rule) — this is
	// the launch-grid reference price for display before that's wired up.
	PriceUAHMinor int `json:"priceUahMinor"`
	// EffectiveMonthlyUAHMinor is 0 for WEEKLY, where §2 says a monthly
	// figure "не застосовується".
	EffectiveMonthlyUAHMinor int `json:"effectiveMonthlyUahMinor"`
	// SavingsVsMonthlyUAHMinor/Percent compare this plan's own period
	// against paying the MONTHLY rate for the same span (0 for WEEKLY and
	// MONTHLY itself, which have no such comparison in §2's table).
	SavingsVsMonthlyUAHMinor int     `json:"savingsVsMonthlyUahMinor"`
	SavingsVsMonthlyPercent  float64 `json:"savingsVsMonthlyPercent"`
	// Recommended marks the paywall's suggested default (§2: "3 місяці —
	// рекомендований стартовий вибір"). Exactly one plan is ever true.
	Recommended bool `json:"recommended"`
}

// RewardedPolicy is the client-facing shape of the rewarded-quest rules —
// display only; the server enforces the same values independently in
// Service.SubmitRewardedView/the Store, never trusting a client replay of
// these numbers.
type RewardedPolicy struct {
	VideoIntervalSeconds   int64 `json:"videoIntervalSeconds"`
	VideosRequired         int   `json:"videosRequired"`
	NominalCompletionHours int   `json:"nominalCompletionHours"`
	QuestWindowSeconds     int64 `json:"questWindowSeconds"`
	RewardSeconds          int64 `json:"rewardSeconds"`
	ClaimCooldownSeconds   int64 `json:"claimCooldownSeconds"`
}

type ReferralMilestone struct {
	QualifiedReferrals int  `json:"qualifiedReferrals"`
	InviterRewardDays  int  `json:"inviterRewardDays"`
	InviteeRewardDays  int  `json:"inviteeRewardDays"`
	Badge              bool `json:"badge"`
}

// Benefit is one entry in the honest feature-availability catalog
// (docs/LINKUP_PLUS_MONETIZATION.md §5/§9: "чесний feature availability
// list із server capability flags"; §9's "Заборонені fake activation ...
// unavailable target features як already included and active"). Available
// is fail-closed: true only once the server actually enforces that
// benefit, never merely because the paywall would like to sell it.
type Benefit struct {
	ID        string `json:"id"`
	Available bool   `json:"available"`
}

type Catalog struct {
	Currency             string              `json:"currency"`
	Plans                []Plan              `json:"plans"`
	RecommendedPlanID    string              `json:"recommendedPlanId"`
	Rewarded             RewardedPolicy      `json:"rewarded"`
	ReferralDeadlineDays int                 `json:"referralDeadlineDays"`
	ReferralMilestones   []ReferralMilestone `json:"referralMilestones"`
	Benefits             []Benefit           `json:"benefits"`
}

type StoreStatus struct {
	PremiumUntil               *time.Time
	VideosWatchedCount         int
	LastVideoWatchedAt         *time.Time
	LastFreePremiumClaimedAt   *time.Time
	QualifiedReferrals         int
	BoundReferralCode          *string
	ReferralQualifyingDeadline *time.Time
}

// Store is the persistence boundary. Every mutating method here owns its
// own transaction/locking internally (matching account.Store's shape) —
// Service never opens a transaction itself, since none of these operations
// span more than one Store call.
type Store interface {
	Status(ctx context.Context, userID string) (StoreStatus, error)
	EnsureReferralCode(ctx context.Context, userID, code string) (string, error)
	BindReferral(ctx context.Context, inviteeID, code string, deadlineDays int) error
	// RecordRewardedView applies one already-provider-verified rewarded ad
	// view: idempotent by receiptHash (a replay returns nil without
	// double-counting), enforces minInterval/questWindow, and — on the 5th
	// verified view of a quest — atomically grants rewardedGrantDuration of
	// premium and starts cooldown.
	RecordRewardedView(ctx context.Context, userID, provider string, receiptHash []byte, now time.Time, minInterval, questWindow, grantDuration, cooldown time.Duration) error
	// RecordPurchase applies one already-provider-verified purchase state.
	// Idempotent by (provider, purchase-identity hash): a replay converges
	// to the same stored state without granting twice. An ACTIVE/GRACE
	// state grants/extends a PAID premium_grants row keyed to this exact
	// purchase; EXPIRED/BILLING_RETRY/REVOKED/REFUNDED retract it. On an
	// ACTIVE transition it also qualifies a pending referral for this user
	// (if any) and awards any newly-reached inviter milestone.
	RecordPurchase(ctx context.Context, userID, provider, productID string, purchaseHash []byte, periodStart, periodEnd time.Time, state string, now time.Time, milestones []ReferralMilestone) error
}

// VerifiedPurchase is the normalized, provider-agnostic result of checking
// a purchase with the store's own server-side verification API. Raw
// purchase tokens are never part of this type by design — only a stable
// PurchaseID the caller hashes for idempotency
// (docs/LINKUP_PLUS_MONETIZATION.md §6/§8.5: "Raw provider receipts/tokens
// не зберігаються").
type VerifiedPurchase struct {
	Provider    string
	ProductID   string
	PurchaseID  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	// State is one of ACTIVE, GRACE, BILLING_RETRY, EXPIRED, REVOKED,
	// REFUNDED — see monetization_subscription_receipts' CHECK constraint
	// and this package's own RecordPurchase doc comment for what each one
	// does to entitlement.
	State string
}

// RewardedVerifier checks a single rewarded-ad-watch event with the ad
// network's own server-side verification (e.g. AdMob SSV/reward callback).
// It must never trust a client-reported "watched=true"
// (docs/LINKUP_PLUS_MONETIZATION.md §6: "client-authoritative watched=true"
// is explicitly forbidden) — this interface exists so Service can enforce
// that boundary regardless of which ad network is plugged in, and so it
// fails closed (ErrCapabilityDisabled) when nothing is configured yet.
type RewardedVerifier interface {
	// VerifyRewardedView returns the ad network's own stable identity for
	// this specific view event (never the raw token/receipt) so the caller
	// can hash it for idempotency.
	VerifyRewardedView(ctx context.Context, userID, rawReceipt string) (provider, verifiedViewID string, err error)
}

// PurchaseVerifier checks a subscription purchase with the store's
// (Google Play/App Store) own server-side verification API. Same
// fail-closed contract as RewardedVerifier.
type PurchaseVerifier interface {
	VerifyPurchase(ctx context.Context, userID, rawPurchaseToken string) (VerifiedPurchase, error)
}

// Capabilities reports which provider-backed actions are actually wired up
// right now — computed from whether a verifier was configured, never a
// static "we plan to support this" flag. README §6.20: "capability flags
// fail closed".
type Capabilities struct {
	PaidVerification      bool `json:"paidVerification"`
	RewardedVerification  bool `json:"rewardedVerification"`
	ReferralQualification bool `json:"referralQualification"`
}

type RewardedStatus struct {
	VideosWatchedCount       int        `json:"videosWatchedCount"`
	NextVideoAt              *time.Time `json:"nextVideoAt,omitempty"`
	LastFreePremiumClaimedAt *time.Time `json:"lastFreePremiumClaimedAt,omitempty"`
	NextFreePremiumClaimAt   *time.Time `json:"nextFreePremiumClaimAt,omitempty"`
}

type ReferralStatus struct {
	ReferralCode       string             `json:"referralCode"`
	BoundReferralCode  *string            `json:"boundReferralCode,omitempty"`
	QualifyingDeadline *time.Time         `json:"qualifyingDeadline,omitempty"`
	QualifiedReferrals int                `json:"qualifiedReferrals"`
	NextMilestone      *ReferralMilestone `json:"nextMilestone,omitempty"`
}

type Status struct {
	PremiumActive bool           `json:"premiumActive"`
	PremiumUntil  *time.Time     `json:"premiumUntil,omitempty"`
	Rewarded      RewardedStatus `json:"rewarded"`
	Referral      ReferralStatus `json:"referral"`
}

type Snapshot struct {
	Catalog      Catalog      `json:"catalog"`
	Status       Status       `json:"status"`
	Capabilities Capabilities `json:"capabilities"`
}
