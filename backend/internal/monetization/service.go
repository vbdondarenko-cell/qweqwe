package monetization

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"
)

const (
	MonthlyPriceUAHMinor    = 14999
	AnnualEffectiveUAHMinor = 9999
	AnnualTotalUAHMinor     = 119988
	AnnualSavingsUAHMinor   = 60000
	RewardedVideosRequired  = 5
	RewardedNominalHours    = 20
	ReferralDeadlineDays    = 14
)

var (
	ErrUnavailable         = errors.New("monetization unavailable")
	ErrInvalidReferralCode = errors.New("invalid referral code")
	ErrReferralAlreadyBound = errors.New("referral already bound")
	ErrSelfReferral        = errors.New("self referral is not allowed")
	ErrReferralExpired     = errors.New("referral deadline expired")
)

type Plan struct {
	ID                       string `json:"id"`
	BillingPeriod            string `json:"billingPeriod"`
	PriceUAHMinor            int    `json:"priceUahMinor"`
	EffectiveMonthlyUAHMinor int    `json:"effectiveMonthlyUahMinor"`
	Total12MonthsUAHMinor    int    `json:"total12MonthsUahMinor"`
}

type RewardedPolicy struct {
	VideoIntervalSeconds   int64 `json:"videoIntervalSeconds"`
	VideosRequired         int   `json:"videosRequired"`
	NominalCompletionHours int   `json:"nominalCompletionHours"`
	RewardSeconds          int64 `json:"rewardSeconds"`
	ClaimCooldownSeconds   int64 `json:"claimCooldownSeconds"`
}

type ReferralMilestone struct {
	QualifiedReferrals int  `json:"qualifiedReferrals"`
	InviterRewardDays  int  `json:"inviterRewardDays"`
	InviteeRewardDays  int  `json:"inviteeRewardDays"`
	Badge              bool `json:"badge"`
}

type Catalog struct {
	Currency                 string              `json:"currency"`
	Plans                    []Plan              `json:"plans"`
	AnnualSavingsUAHMinor    int                 `json:"annualSavingsUahMinor"`
	AnnualSavingsPercent     float64             `json:"annualSavingsPercent"`
	Rewarded                 RewardedPolicy      `json:"rewarded"`
	ReferralDeadlineDays     int                 `json:"referralDeadlineDays"`
	ReferralMilestones       []ReferralMilestone `json:"referralMilestones"`
}

type StoreStatus struct {
	PremiumUntil             *time.Time
	VideosWatchedCount       int
	LastVideoWatchedAt       *time.Time
	LastFreePremiumClaimedAt *time.Time
	QualifiedReferrals       int
	BoundReferralCode        *string
	ReferralQualifyingDeadline *time.Time
}

type Store interface {
	Status(ctx context.Context, userID string) (StoreStatus, error)
	EnsureReferralCode(ctx context.Context, userID, code string) (string, error)
	BindReferral(ctx context.Context, inviteeID, code string, deadlineDays int) error
}

type Capabilities struct {
	PaidVerification       bool `json:"paidVerification"`
	RewardedVerification   bool `json:"rewardedVerification"`
	ReferralQualification  bool `json:"referralQualification"`
}

type RewardedStatus struct {
	VideosWatchedCount       int        `json:"videosWatchedCount"`
	NextVideoAt              *time.Time `json:"nextVideoAt,omitempty"`
	LastFreePremiumClaimedAt *time.Time `json:"lastFreePremiumClaimedAt,omitempty"`
	NextFreePremiumClaimAt   *time.Time `json:"nextFreePremiumClaimAt,omitempty"`
}

type ReferralStatus struct {
	ReferralCode              string             `json:"referralCode"`
	BoundReferralCode         *string            `json:"boundReferralCode,omitempty"`
	QualifyingDeadline        *time.Time          `json:"qualifyingDeadline,omitempty"`
	QualifiedReferrals        int                 `json:"qualifiedReferrals"`
	NextMilestone             *ReferralMilestone  `json:"nextMilestone,omitempty"`
}

type Status struct {
	PremiumActive bool           `json:"premiumActive"`
	PremiumUntil  *time.Time     `json:"premiumUntil,omitempty"`
	Rewarded      RewardedStatus `json:"rewarded"`
	Referral      ReferralStatus `json:"referral"`
}

type Snapshot struct {
	Catalog       Catalog      `json:"catalog"`
	Status        Status       `json:"status"`
	Capabilities Capabilities `json:"capabilities"`
}

type Service struct {
	store        Store
	now          func() time.Time
	capabilities Capabilities
}

func NewService(store Store) *Service {
	return &Service{store: store, now: time.Now}
}

func (s *Service) Catalog() Catalog {
	return Catalog{
		Currency: "UAH",
		Plans: []Plan{
			{ID: "monthly", BillingPeriod: "MONTH", PriceUAHMinor: MonthlyPriceUAHMinor, EffectiveMonthlyUAHMinor: MonthlyPriceUAHMinor, Total12MonthsUAHMinor: 179988},
			{ID: "annual", BillingPeriod: "YEAR", PriceUAHMinor: AnnualTotalUAHMinor, EffectiveMonthlyUAHMinor: AnnualEffectiveUAHMinor, Total12MonthsUAHMinor: AnnualTotalUAHMinor},
		},
		AnnualSavingsUAHMinor: AnnualSavingsUAHMinor,
		AnnualSavingsPercent:  33.3,
		Rewarded: RewardedPolicy{
			VideoIntervalSeconds:   int64((4 * time.Hour) / time.Second),
			VideosRequired:         RewardedVideosRequired,
			NominalCompletionHours: RewardedNominalHours,
			RewardSeconds:          int64((24 * time.Hour) / time.Second),
			ClaimCooldownSeconds:   int64((7 * 24 * time.Hour) / time.Second),
		},
		ReferralDeadlineDays: ReferralDeadlineDays,
		ReferralMilestones:   referralMilestones(),
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
		next := raw.LastVideoWatchedAt.Add(4 * time.Hour)
		status.Rewarded.NextVideoAt = &next
	}
	if raw.LastFreePremiumClaimedAt != nil {
		next := raw.LastFreePremiumClaimedAt.Add(7 * 24 * time.Hour)
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
