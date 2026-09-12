// Package guardian implements README §6.17's "Ghost Guardian": a
// temporary, opaque, revocable link a Slot participant can hand to a
// trusted person (who need not have a LinkUp account at all) so that
// person can check whether the participant is still safely checked in,
// without exposing the Slot's location, title, or roster.
//
// Only Mode STATUS_ONLY is implemented. README §6.17 also names
// ETA_APPROXIMATE, SAFETY_RADAR and LIVE_PRECISE, but all three depend on
// a live-location-submission pipeline this repo doesn't have and isn't
// specified (how often the client would push a position, what payload,
// what retention, and — for LIVE_PRECISE specifically — the "окрема явна
// згода" consent flow the LinkUp+ monetization contract requires for the
// analogous Squad Radar feature). Service.CreateLink rejects every other
// mode outright with ErrModeNotSupported rather than accepting the
// request and silently only ever behaving like STATUS_ONLY — a client
// asking for LIVE_PRECISE must get a clear, fail-closed "not yet", never a
// link that quietly never delivers what it promised.
package guardian

import (
	"context"
	"errors"
	"time"
)

type Mode string

const (
	ModeStatusOnly     Mode = "STATUS_ONLY"
	ModeETAApproximate Mode = "ETA_APPROXIMATE"
	ModeSafetyRadar    Mode = "SAFETY_RADAR"
	ModeLivePrecise    Mode = "LIVE_PRECISE"
)

const (
	MinTTL = 5 * time.Minute
	MaxTTL = 24 * time.Hour
)

var (
	ErrInvalidInput     = errors.New("invalid guardian link input")
	ErrForbidden        = errors.New("guardian link forbidden")
	ErrModeNotSupported = errors.New("guardian link mode is not supported yet")
	// ErrLinkUnavailable is returned uniformly whether a token never
	// existed, has expired, or was revoked — README §6.17's own
	// "indistinguishable/burned access after expiry/revoke/completion"
	// requirement. It never carries a hint about which of those applied.
	ErrLinkUnavailable = errors.New("guardian link is unavailable")
)

// SlotStatus is the entirety of what a valid STATUS_ONLY link reveals.
type SlotStatus string

const (
	// SlotStatusActive covers every Slot lifecycle state where the
	// participant could plausibly still be checked in (PUBLISHED, FILLING,
	// FULL, ACTIVE).
	SlotStatusActive SlotStatus = "ACTIVE"
	// SlotStatusEnded is the fallback for every other state (COMPLETED,
	// CANCELLED, EXPIRED, MODERATED, DRAFT) — a valid, non-revoked link
	// whose underlying Slot has simply run its course.
	SlotStatusEnded SlotStatus = "ENDED"
)

// Link is what CreateLink persists.
type Link struct {
	ID        string
	SlotID    string
	CreatedBy string
	Mode      Mode
	CreatedAt time.Time
	ExpiresAt time.Time
}

// LinkView is CreateLink's return value — the only moment the raw token is
// ever available. Only its hash is ever stored (matching how session and
// password-reset tokens already work in this codebase), so it can never be
// retrieved again after this call returns.
type LinkView struct {
	Link
	Token string
}

// Status is what Service.Access returns to whoever holds a valid token.
type Status struct {
	SharerDisplayName string     `json:"sharerDisplayName"`
	SlotStatus        SlotStatus `json:"slotStatus"`
}

type Store interface {
	// CreateLink authorizes actorID against slotID (the Slot's host or a
	// currently-accepted member) before creating the link. Returns
	// ErrForbidden uniformly whether the Slot doesn't exist or actorID
	// simply isn't on it — never distinguishing the two.
	CreateLink(ctx context.Context, id, slotID, actorID string, mode Mode, tokenHash []byte, now, expiresAt time.Time) error
	// Revoke marks the link revoked only if actorID is its own creator.
	// Returns ErrForbidden uniformly whether the link doesn't exist or
	// belongs to someone else.
	Revoke(ctx context.Context, linkID, actorID string, now time.Time) error
	// Access resolves a token hash to a Status, or ErrLinkUnavailable
	// uniformly for a token that never existed, is expired, or was
	// revoked.
	Access(ctx context.Context, tokenHash []byte, now time.Time) (Status, error)
}
