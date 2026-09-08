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
	ErrNoLocality         = errors.New("no locality contains observation")
	ErrNotFound           = errors.New("city context not found")
	ErrResolverUnavailable = errors.New("city context resolver unavailable")
)

type Policy struct {
	PreciseMaxAge       time.Duration
	ApproximateMaxAge   time.Duration
	PreciseMaxAccuracyM int
	ApproximateMaxAccuracyM int
	MaxFutureSkew       time.Duration
	LockTTL             time.Duration
	SwitchWindow        time.Duration
	SwitchConfirmations int
}

func DefaultPolicy() Policy {
	return Policy{
		PreciseMaxAge: 2 * time.Minute,
		ApproximateMaxAge: 10 * time.Minute,
		PreciseMaxAccuracyM: 500,
		ApproximateMaxAccuracyM: 5000,
		MaxFutureSkew: 30 * time.Second,
		LockTTL: 6 * time.Hour,
		SwitchWindow: 10 * time.Minute,
		SwitchConfirmations: 2,
	}
}

func (p Policy) Valid() bool {
	return p.PreciseMaxAge > 0 && p.ApproximateMaxAge > 0 &&
		p.PreciseMaxAccuracyM > 0 && p.ApproximateMaxAccuracyM > 0 &&
		p.MaxFutureSkew >= 0 && p.LockTTL > 0 && p.SwitchWindow > 0 && p.SwitchConfirmations >= 2
}

type Observation struct {
	LatitudeE6      int
	LongitudeE6     int
	AccuracyM       int
	PermissionClass PermissionClass
	CapturedAt      time.Time
	Mocked          bool
}

func (o Observation) Valid(now time.Time, policy Policy) bool {
	if !policy.Valid() || o.Mocked || o.LatitudeE6 < -90000000 || o.LatitudeE6 > 90000000 ||
		o.LongitudeE6 < -180000000 || o.LongitudeE6 > 180000000 || o.AccuracyM <= 0 || o.CapturedAt.IsZero() {
		return false
	}
	captured := o.CapturedAt.UTC()
	now = now.UTC()
	if captured.After(now.Add(policy.MaxFutureSkew)) {
		return false
	}
	age := now.Sub(captured)
	if age < -policy.MaxFutureSkew {
		return false
	}
	switch o.PermissionClass {
	case PermissionPrecise:
		return age <= policy.PreciseMaxAge && o.AccuracyM <= policy.PreciseMaxAccuracyM
	case PermissionApproximate:
		return age <= policy.ApproximateMaxAge && o.AccuracyM <= policy.ApproximateMaxAccuracyM
	default:
		return false
	}
}

type Locality struct {
	ID                  string `json:"id"`
	Name                string `json:"name"`
	CountryCode         string `json:"countryCode"`
	Timezone            string `json:"timezone"`
	CentroidLatitudeE6  int    `json:"centroidLatitudeE6"`
	CentroidLongitudeE6 int    `json:"centroidLongitudeE6"`
}

func (l Locality) Valid() bool {
	return validUUID(l.ID) && strings.TrimSpace(l.Name) != "" && len(l.CountryCode) == 2 &&
		strings.ToUpper(l.CountryCode) == l.CountryCode && strings.TrimSpace(l.Timezone) != "" &&
		l.CentroidLatitudeE6 >= -90000000 && l.CentroidLatitudeE6 <= 90000000 &&
		l.CentroidLongitudeE6 >= -180000000 && l.CentroidLongitudeE6 <= 180000000
}

type Context struct {
	Locality        Locality        `json:"locality"`
	PermissionClass PermissionClass `json:"permissionClass"`
	AccuracyM       int             `json:"accuracyM"`
	ObservedAt      time.Time       `json:"observedAt"`
	ExpiresAt       time.Time       `json:"expiresAt"`
	SwitchPending   bool            `json:"switchPending"`
}

type Lock struct {
	UserID                  string
	Locality                Locality
	PermissionClass         PermissionClass
	AccuracyM               int
	ObservedAt              time.Time
	ExpiresAt               time.Time
	CandidateLocalityID     *string
	CandidateCount          int
	CandidateObservedAt     *time.Time
}

func (l Lock) Public() Context {
	return Context{
		Locality: l.Locality,
		PermissionClass: l.PermissionClass,
		AccuracyM: l.AccuracyM,
		ObservedAt: l.ObservedAt,
		ExpiresAt: l.ExpiresAt,
		SwitchPending: l.CandidateLocalityID != nil && l.CandidateCount > 0,
	}
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
		return nil, ErrResolverUnavailable
	}
	return &Service{store: store, policy: policy, now: time.Now}, nil
}

func (s *Service) Resolve(ctx context.Context, userID string, observation Observation) (Context, error) {
	userID = strings.TrimSpace(userID)
	now := s.now().UTC()
	if userID == "" || !observation.Valid(now, s.policy) {
		return Context{}, ErrInvalidObservation
	}
	return s.store.Apply(ctx, userID, observation, s.policy, now)
}

func (s *Service) Current(ctx context.Context, userID string) (Context, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Context{}, ErrNotFound
	}
	return s.store.Current(ctx, userID, s.now().UTC())
}

func NextLock(current *Lock, resolved Locality, observation Observation, now time.Time, policy Policy) (Lock, error) {
	if !resolved.Valid() || !observation.Valid(now, policy) {
		return Lock{}, ErrInvalidObservation
	}
	now = now.UTC()
	freshExpiry := now.Add(policy.LockTTL)
	if current == nil || current.ExpiresAt.Before(now) || current.ExpiresAt.Equal(now) {
		return Lock{
			Locality: resolved, PermissionClass: observation.PermissionClass, AccuracyM: observation.AccuracyM,
			ObservedAt: observation.CapturedAt.UTC(), ExpiresAt: freshExpiry,
		}, nil
	}
	if current.Locality.ID == resolved.ID {
		next := *current
		next.Locality = resolved
		next.PermissionClass = observation.PermissionClass
		next.AccuracyM = observation.AccuracyM
		next.ObservedAt = observation.CapturedAt.UTC()
		next.ExpiresAt = freshExpiry
		next.CandidateLocalityID = nil
		next.CandidateCount = 0
		next.CandidateObservedAt = nil
		return next, nil
	}

	next := *current
	candidateStillFresh := current.CandidateLocalityID != nil && *current.CandidateLocalityID == resolved.ID &&
		current.CandidateObservedAt != nil && now.Sub(current.CandidateObservedAt.UTC()) <= policy.SwitchWindow
	if candidateStillFresh {
		next.CandidateCount++
	} else {
		id := resolved.ID
		next.CandidateLocalityID = &id
		next.CandidateCount = 1
	}
	observed := observation.CapturedAt.UTC()
	next.CandidateObservedAt = &observed
	if next.CandidateCount >= policy.SwitchConfirmations {
		next.Locality = resolved
		next.PermissionClass = observation.PermissionClass
		next.AccuracyM = observation.AccuracyM
		next.ObservedAt = observed
		next.ExpiresAt = freshExpiry
		next.CandidateLocalityID = nil
		next.CandidateCount = 0
		next.CandidateObservedAt = nil
	}
	return next, nil
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i, c := range []byte(value) {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
