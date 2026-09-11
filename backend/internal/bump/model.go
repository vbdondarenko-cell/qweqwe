// Package bump implements README §6.9's BUMP proof baseline: mutual,
// server-verified proof that two accepted participants of the same Slot
// were both physically present together. It never trusts a single client
// claim — reliability only moves once both sides of a pair have
// independently submitted a matching claim (see Store.Confirm and its
// postgres implementation for the exact mechanics).
package bump

import (
	"context"
	"errors"
	"strings"
	"time"
)

var (
	ErrInvalidInput     = errors.New("invalid bump input")
	ErrNotFound         = errors.New("bump slot not found")
	ErrForbidden        = errors.New("bump forbidden")
	ErrNotEligible      = errors.New("bump not eligible")
	ErrInvalidChallenge = errors.New("bump challenge invalid or expired")
)

// Band is the coarse, public-facing reliability signal (README §6.9:
// "private/public reliability bands"). VerifiedBumpCount on Reliability is
// the private exact figure a user sees for themselves; Band is what other
// surfaces may show about a user without exposing the precise count.
type Band string

const (
	BandNew      Band = "NEW"
	BandBuilding Band = "BUILDING"
	BandReliable Band = "RELIABLE"
	BandTrusted  Band = "TRUSTED"
)

// ComputeBand maps a verified mutual BUMP count to a public reliability
// band. README §6.9 requires bands to exist and to be earned only through
// server-verified mutual proof; it does not specify exact cutoffs. These
// thresholds are a conservative initial policy, stated here rather than
// hidden, and are a product-tuning decision to revisit later — not a claim
// that they are final.
func ComputeBand(verifiedCount int) Band {
	switch {
	case verifiedCount >= 10:
		return BandTrusted
	case verifiedCount >= 3:
		return BandReliable
	case verifiedCount >= 1:
		return BandBuilding
	default:
		return BandNew
	}
}

// Challenge is a server-issued, single-use anti-replay token: a client must
// hold a fresh, unexpired Challenge for (slot, actor) before Confirm will
// accept a claim from that actor. It is the foundation-block stand-in for
// README's "Android Keystore-backed device proof", which requires an
// Android build this environment cannot produce or verify — see
// IMPLEMENTATION_STATUS.md for that explicit gap.
type Challenge struct {
	Nonce     string
	SlotID    string
	ExpiresAt time.Time
}

// ConfirmResult reports whether THIS call completed the mutual pair.
// Verified is false when only one direction exists yet — the caller's own
// claim was recorded, but nothing about their or the counterpart's
// reliability changed (README: "Client tap alone can never increase
// trust/reliability").
type ConfirmResult struct {
	Verified    bool
	ConfirmedAt time.Time
}

type Reliability struct {
	UserID            string `json:"userId"`
	VerifiedBumpCount int    `json:"verifiedBumpCount"`
	Band              Band   `json:"band"`
}

type VaultEntry struct {
	SlotID        string    `json:"slotId"`
	SlotTitle     string    `json:"slotTitle"`
	CounterpartID string    `json:"counterpartUserId"`
	ConfirmedAt   time.Time `json:"confirmedAt"`
}

// Store is implemented by the postgres package.
type Store interface {
	// IssueChallenge fails with ErrNotFound if the Slot does not exist, and
	// ErrNotEligible if the actor is not currently a host/accepted
	// participant of a Slot in ACTIVE or COMPLETED state.
	IssueChallenge(ctx context.Context, actorID, slotID string, ttl time.Duration) (Challenge, error)
	// Confirm fails with ErrInvalidChallenge if nonce is unknown, expired,
	// already consumed, or does not belong to (actorID, slotID);
	// ErrNotEligible if either side is not a host/accepted participant of
	// that Slot; ErrForbidden if the pair is blocked either direction.
	Confirm(ctx context.Context, actorID, slotID, counterpartID, nonce string) (ConfirmResult, error)
	Reliability(ctx context.Context, userID string) (Reliability, error)
	Vault(ctx context.Context, userID string, limit int) ([]VaultEntry, error)
	// PublicBand fails with ErrForbidden if viewerID and targetUserID have
	// blocked each other in either direction. It never returns the exact
	// VerifiedBumpCount — see PublicBand's own doc comment on Service for
	// why that split exists.
	PublicBand(ctx context.Context, viewerID, targetUserID string) (Band, error)
}

type Service struct {
	store        Store
	challengeTTL time.Duration
}

func NewService(store Store, challengeTTL time.Duration) (*Service, error) {
	if store == nil || challengeTTL <= 0 {
		return nil, ErrInvalidInput
	}
	return &Service{store: store, challengeTTL: challengeTTL}, nil
}

func (s *Service) IssueChallenge(ctx context.Context, actorID, slotID string) (Challenge, error) {
	actorID, slotID = strings.TrimSpace(actorID), strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" {
		return Challenge{}, ErrInvalidInput
	}
	return s.store.IssueChallenge(ctx, actorID, slotID, s.challengeTTL)
}

func (s *Service) Confirm(ctx context.Context, actorID, slotID, counterpartID, nonce string) (ConfirmResult, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	counterpartID = strings.TrimSpace(counterpartID)
	nonce = strings.TrimSpace(nonce)
	if actorID == "" || slotID == "" || counterpartID == "" || nonce == "" {
		return ConfirmResult{}, ErrInvalidInput
	}
	if actorID == counterpartID {
		return ConfirmResult{}, ErrInvalidInput
	}
	return s.store.Confirm(ctx, actorID, slotID, counterpartID, nonce)
}

func (s *Service) Reliability(ctx context.Context, userID string) (Reliability, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return Reliability{}, ErrInvalidInput
	}
	return s.store.Reliability(ctx, userID)
}

// PublicBand closes README §6.9's "private/public reliability bands" split
// at the API surface, not just in the domain model: GetReliability (§48)
// only ever let a user read their own full Reliability (exact
// VerifiedBumpCount included), so there was previously no way for anyone
// to see ANOTHER user's public-facing Band at all — the private/public
// distinction the Band type's own doc comment describes existed in code
// but was never actually exposed as two different surfaces. This is the
// public one: it returns only the coarse Band, never the exact count,
// and is subject to the same block relationship every other
// cross-user-visible surface in this codebase already enforces.
func (s *Service) PublicBand(ctx context.Context, viewerID, targetUserID string) (Band, error) {
	viewerID = strings.TrimSpace(viewerID)
	targetUserID = strings.TrimSpace(targetUserID)
	if viewerID == "" || targetUserID == "" {
		return "", ErrInvalidInput
	}
	return s.store.PublicBand(ctx, viewerID, targetUserID)
}

func (s *Service) Vault(ctx context.Context, userID string, limit int) ([]VaultEntry, error) {
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return nil, ErrInvalidInput
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	return s.store.Vault(ctx, userID, limit)
}
