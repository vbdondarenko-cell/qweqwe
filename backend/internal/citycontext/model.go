package citycontext

import (
	"context"
	"errors"
	"strings"
	"time"
)

type PermissionClass string

const (
	PermissionApproximate PermissionClass = "APPROXIMATE"
	PermissionPrecise     PermissionClass = "PRECISE"
)

var (
	ErrInvalidObservation = errors.New("invalid city context observation")
	ErrNoLocality         = errors.New("no locality resolved")
	ErrNotFound           = errors.New("city context not found")
	ErrResolverUnavailable = errors.New("city context resolver unavailable")
)

type Policy struct {
	PreciseMaxAge       time.Duration
	ApproximateMaxAge   time.Duration
	MaxFutureSkew       time.Duration
	PreciseMaxAccuracyM int
	ApproxMaxAccuracyM  int
	LockTTL             time.Duration
	SwitchWindow        time.Duration
	SwitchConfirmations int
}

func DefaultPolicy() Policy {
	return Policy{
		PreciseMaxAge:       2 * time.Minute,
		ApproximateMaxAge:   10 * time.Minute,
		MaxFutureSkew:       30 * time.Second,
		PreciseMaxAccuracyM: 500,
		ApproxMaxAccuracyM:  5_000,
		LockTTL:             6 * time.Hour,
		SwitchWindow:        10 * time.Minute,
		SwitchConfirmations: 2,
	}
}

func (p Policy) Valid() bool {
	return p.PreciseMaxAge > 0 && p.ApproximateMaxAge > 0 && p.MaxFutureSkew >= 0 &&
		p.PreciseMaxAccuracyM > 0 && p.ApproxMaxAccuracyM >= p.PreciseMaxAccuracyM &&
		p.LockTTL > 0 && p.SwitchWindow > 0 && p.SwitchConfirmations >= 2 && p.SwitchConfirmations <= 10
}

type Observation struct {
	LatitudeE6      int
	LongitudeE6     int
	AccuracyM       int
	PermissionClass PermissionClass
	CapturedAt      time.Time
	Mocked           bool
}

func (o Observation) Valid(now time.Time, policy Policy) bool {
	if !policy.Valid() || now.IsZero() || o.Mocked || o.CapturedAt.IsZero() ||
		o.LatitudeE6 < -90_000_000 || o.LatitudeE6 > 90_000_000 ||
		o.LongitudeE6 < -180_000_000 || o.LongitudeE6 > 180_000_000 || o.AccuracyM < 1 {
		return false
	}
	if o.CapturedAt.After(now.Add(policy.MaxFutureSkew)) {
		return false
	}
	age := now.Sub(o.CapturedAt)
	if age < -policy.MaxFutureSkew {
		return false
	}
	switch o.PermissionClass {
	case PermissionPrecise:
		return age <= policy.PreciseMaxAge && o.AccuracyM <= policy.PreciseMaxAccuracyM
	case PermissionApproximate:
		return age <= policy.ApproximateMaxAge && o.AccuracyM <= policy.ApproxMaxAccuracyM
	default:
		return false
	}
}

type Locality struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	CountryCode string `json:"countryCode"`
	Timezone    string `json:"timezone"`
}

func (l Locality) Valid() bool {
	return strings.TrimSpace(l.ID) != "" && strings.TrimSpace(l.Name) != "" &&
		len(l.CountryCode) == 2 && strings.TrimSpace(l.Timezone) != ""
}

type Lock struct {
	UserID              string
	Locality             Locality
	PermissionClass      PermissionClass
	AccuracyM            int
	ObservedAt           time.Time
	ExpiresAt            time.Time
	CandidateLocalityID  *string
	CandidateCount       int
	CandidateObservedAt  *time.Time
}

type Context struct {
	Locality        Locality        `json:"locality"`
	PermissionClass PermissionClass `json:"permissionClass"`
	AccuracyM       int             `json:"accuracyM"`
	ObservedAt      time.Time       `json:"observedAt"`
	ExpiresAt       time.Time       `json:"expiresAt"`
	SwitchPending   bool            `json:"switchPending"`
}

func (l Lock) Public() Context {
	return Context{
		Locality: l.Locality, PermissionClass: l.PermissionClass, AccuracyM: l.AccuracyM,
		ObservedAt: l.ObservedAt, ExpiresAt: l.ExpiresAt, SwitchPending: l.CandidateLocalityID != nil,
	}
}

func NextLock(current *Lock, resolved Locality, observation Observation, now time.Time, policy Policy) (Lock, error) {
	if !resolved.Valid() || !observation.Valid(now, policy) {
		return Lock{}, ErrInvalidObservation
	}

	fresh := func() Lock {
		return Lock{
			Locality: resolved,
			PermissionClass: observation.PermissionClass,
			AccuracyM: observation.AccuracyM,
			ObservedAt: observation.CapturedAt.UTC(),
			ExpiresAt: observation.CapturedAt.UTC().Add(policy.LockTTL),
		}
	}
	if current == nil || !now.Before(current.ExpiresAt) {
		return fresh(), nil
	}
	if current.Locality.ID == resolved.ID {
		next := fresh()
		next.UserID = current.UserID
		return next, nil
	}

	next := *current
	candidateCount := 1
	candidateAt := observation.CapturedAt.UTC()
	if current.CandidateLocalityID != nil && *current.CandidateLocalityID == resolved.ID &&
		current.CandidateObservedAt != nil && !observation.CapturedAt.Before(*current.CandidateObservedAt) &&
		observation.CapturedAt.Sub(*current.CandidateObservedAt) <= policy.SwitchWindow {
		candidateCount = current.CandidateCount + 1
	}
	if candidateCount >= policy.SwitchConfirmations {
		switched := fresh()
		switched.UserID = current.UserID
		return switched, nil
	}
	next.CandidateLocalityID = &resolved.ID
	next.CandidateCount = candidateCount
	next.CandidateObservedAt = &candidateAt
	return next, nil
}

type Store interface {
	Apply(ctx context.Context, userID string, observation Observation, policy Policy, now time.Time) (Context, error)
	Current(ctx context.Context, userID string, now time.Time) (Context, error)
}

type Service struct {
	store  Store
	policy Policy
	now    func() time.Time
}

func NewService(store Store, policy Policy) (*Service, error) {
	if store == nil || !policy.Valid() {
		return nil, ErrInvalidObservation
	}
	return &Service{store: store, policy: policy, now: func() time.Time { return time.Now().UTC() }}, nil
}

func (s *Service) Resolve(ctx context.Context, userID string, observation Observation) (Context, error) {
	now := s.now()
	if strings.TrimSpace(userID) == "" || !observation.Valid(now, s.policy) {
		return Context{}, ErrInvalidObservation
	}
	return s.store.Apply(ctx, userID, observation, s.policy, now)
}

func (s *Service) Current(ctx context.Context, userID string) (Context, error) {
	if strings.TrimSpace(userID) == "" {
		return Context{}, ErrInvalidObservation
	}
	return s.store.Current(ctx, userID, s.now())
}
