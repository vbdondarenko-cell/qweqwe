package slot

import (
	"context"
	"errors"
	"time"
)

type State string
type AccessMode string
type Visibility string
type ViewerState string

const (
	StateDraft     State = "DRAFT"
	StatePublished State = "PUBLISHED"
	StateFilling   State = "FILLING"
	StateFull      State = "FULL"
	StateActive    State = "ACTIVE"
	StateCompleted State = "COMPLETED"
	StateCancelled State = "CANCELLED"
	StateExpired   State = "EXPIRED"
	StateModerated State = "MODERATED"

	AccessInstant  AccessMode = "INSTANT"
	AccessApproval AccessMode = "APPROVAL"
	AccessWaitlist AccessMode = "WAITLIST"

	VisibilityPublic Visibility = "PUBLIC"
	// VisibilityPrivate is the first non-PUBLIC visibility mode README
	// §4.3 lists (v1.0 mandates Public; wider visibility is v1.1 scope).
	// Every existing discovery surface (Pulse, Map, the Get "stranger"
	// branch, realtime lifecycle-event visibility) already gates on
	// visibility='PUBLIC' explicitly, so a PRIVATE Slot is automatically
	// excluded from all of them with no further change. It is still
	// directly reachable by anyone who already has a relationship (host,
	// accepted member, or a live pending request) or who knows the Slot ID
	// and calls Request/Join directly — Request/Join never check
	// visibility, matching an "invite by sharing the ID" model. The other
	// five modes README §4.3 lists (LINKS/SELECTED/CITY/LASSO/
	// TRAVEL_CORRIDOR) — LINKS and SELECTED are now also implemented, see
	// their own constants below; CITY/LASSO/TRAVEL_CORRIDOR remain
	// unimplemented; see internal/slot's own validation for the closed set
	// this API actually accepts.
	VisibilityPrivate Visibility = "PRIVATE"
	// VisibilityLinks is README §4.3's "Friends/Links" mode: discoverable
	// only to the host's accepted friends (internal/friend.Store.AreFriends),
	// same "discoverability gate, not access-control gate" design as
	// VisibilityPrivate — anyone with the Slot ID can still Request/Join
	// regardless of visibility, matching every other mode's contract. The
	// friendship check itself lives in the postgres query layer (V11SlotStore),
	// since the Slot domain has no dependency on internal/friend and this
	// block does not introduce one; see internal/postgres/v11_slot_store.go.
	VisibilityLinks Visibility = "LINKS"
	// VisibilitySelected is README §4.3's "Selected people" mode: the host
	// picks a specific, explicit allow-list of individual users at creation
	// time (CreateInput.SelectedUserIDs), distinct from LINKS's "anyone who
	// is a real mutual friend" rule. Same discoverability-gate contract as
	// every other mode: Request/Join never check it, so a stranger handed
	// the Slot ID directly can still reach it regardless of the allow-list.
	// The allow-list itself lives in postgres (slot_selected_viewers,
	// migration 000032); it can be replaced wholesale after creation via
	// EditInput.SelectedUserIDs while the Slot is still DRAFT.
	VisibilitySelected Visibility = "SELECTED"

	ViewerNone     ViewerState = "NONE"
	ViewerPending  ViewerState = "PENDING"
	ViewerAccepted ViewerState = "ACCEPTED"
	ViewerHost     ViewerState = "HOST"

	MaxV1Capacity      = 100
	MaxPendingRequests = 100
)

var (
	ErrNotFound            = errors.New("slot not found")
	ErrForbidden           = errors.New("slot forbidden")
	ErrConflict            = errors.New("slot version conflict")
	ErrInvalidInput        = errors.New("invalid slot input")
	ErrInvalidState        = errors.New("invalid slot state")
	ErrIdempotencyConflict = errors.New("idempotency key conflict")
	ErrCapacityFull        = errors.New("slot capacity full")
	ErrRequestLimit        = errors.New("slot pending request limit reached")
	ErrDuplicateRequest    = errors.New("duplicate slot request")
	ErrAlreadyMember       = errors.New("already accepted member")
	ErrRequestNotFound     = errors.New("slot request not found")
)

type Organizer struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
}

type Slot struct {
	ID               string     `json:"id"`
	Organizer        Organizer  `json:"organizer"`
	Title            string     `json:"title"`
	Activity         string     `json:"activity"`
	Details          *string    `json:"details,omitempty"`
	PlaceText        string     `json:"placeText"`
	ZoneText         *string    `json:"zoneText,omitempty"`
	CanonicalPlaceID *string    `json:"canonicalPlaceId,omitempty"`
	StartAt          *time.Time `json:"startAt,omitempty"`
	Capacity         int        `json:"capacity"`
	AcceptedCount    int        `json:"acceptedCount"`
	State            State      `json:"state"`
	AccessMode       AccessMode `json:"accessMode"`
	Visibility       Visibility `json:"visibility"`
	// SelectedUserIDs is populated on Create/Edit/Get/ListPulse/ListMine for
	// a VisibilitySelected Slot, but only for the Slot's own host — it is
	// the host's private curation list, not a roster anyone selected (or
	// anyone else who reaches the Slot) is shown. Always empty for every
	// other viewer and every other visibility.
	SelectedUserIDs []string    `json:"selectedUserIds,omitempty"`
	ViewerState     ViewerState `json:"viewerState"`
	Version         int64       `json:"version"`
	CreatedAt       time.Time   `json:"createdAt"`
	UpdatedAt       time.Time   `json:"updatedAt"`
}

type PendingRequest struct {
	User        Organizer `json:"user"`
	RequestedAt time.Time `json:"requestedAt"`
	// QueuePosition is non-nil only when the Slot's access mode is
	// WAITLIST: the requester's 1-based FIFO position among all pending
	// requests for that Slot, in the same order promoteOldestWaitlistTx
	// promotes from (created_at ASC, user_id ASC as a deterministic
	// tie-breaker). Always nil for APPROVAL mode, which has no queue —
	// only a host-decides pending list.
	QueuePosition *int `json:"queuePosition,omitempty"`
	// Expired reports whether this WAITLIST queue position has passed the
	// server's waitlistRequestTTL and will not be promoted until the
	// requester requests again (README §6.6 request-expiry hardening;
	// mirrors the same expiry already applied to Get/ListPulse/ListMine/
	// Map/realtime visibility). Always false for APPROVAL mode, which has
	// no expiry concept for a pending request.
	Expired bool `json:"expired"`
}

type CreateInput struct {
	Title            string
	Activity         string
	Details          *string
	PlaceText        string
	ZoneText         *string
	CanonicalPlaceID *string
	StartAt          *time.Time
	Capacity         int
	AccessMode       *AccessMode
	// Visibility is nil-safe (defaults to VisibilityPublic, v1.0's only
	// supported value); the v1.1 hosting draft/publish path additionally
	// accepts VisibilityPrivate. Like AccessMode, it can also be changed
	// later via EditInput.Visibility, but (again matching AccessMode) only
	// while the Slot is still DRAFT — see EditInput.Visibility's comment.
	Visibility *Visibility
	// SelectedUserIDs is required (non-empty) when Visibility is
	// VisibilitySelected, ignored otherwise. Each entry must be a valid
	// UUID; duplicates are silently deduplicated by normalizeCreate.
	SelectedUserIDs []string
}

type EditInput struct {
	ExpectedVersion       int64
	Title                 *string
	Details               *string
	PlaceText             *string
	ZoneText              *string
	CanonicalPlaceID      *string
	ClearCanonicalPlaceID bool
	StartAt               *time.Time
	ClearStartAt          bool
	Capacity              *int
	AccessMode            *AccessMode
	// Visibility, like AccessMode, is only honored by the v1.1 store while
	// the Slot is still DRAFT (ErrInvalidState otherwise) — once a Slot is
	// published, changing who can discover it is a bigger decision (it can
	// silently move already-visible strangers into an "invite only by
	// shared ID" state) than this block takes on; a host who needs to
	// change visibility after publishing must cancel and recreate.
	Visibility *Visibility
	// SelectedUserIDs is read only when Visibility is non-nil and equal to
	// VisibilitySelected (still DRAFT-only, same gate as Visibility
	// itself). It always fully replaces any existing allow-list — there is
	// no partial add/remove — and is validated the same way
	// CreateInput.SelectedUserIDs is (non-empty, well-formed UUIDs,
	// deduplicated). This closes the previous "cancel and recreate" gap
	// for both switching a DRAFT Slot into SELECTED and re-picking an
	// already-SELECTED DRAFT Slot's allow-list. Ignored for every other
	// Visibility value, and ignored entirely when Visibility is nil (an
	// Edit that only touches other fields never touches the allow-list).
	SelectedUserIDs []string
}

type Store interface {
	RemoveMember(ctx context.Context, actorID, slotID, memberID string, expectedVersion int64, key string, requestHash []byte, now time.Time) (Slot, error)
	ListAccepted(ctx context.Context, actorID, slotID string) ([]Organizer, error)
	ListMine(ctx context.Context, actorID, view string, limit int) ([]Slot, error)
	Create(ctx context.Context, actorID string, candidate Slot, idempotencyKey string, requestHash []byte) (Slot, error)
	Get(ctx context.Context, actorID, slotID string) (Slot, error)
	ListPulse(ctx context.Context, actorID string, limit int) ([]Slot, error)
	Edit(ctx context.Context, actorID, slotID string, patch EditInput, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Cancel(ctx context.Context, actorID, slotID string, expectedVersion int64, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Request(ctx context.Context, actorID, slotID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Leave(ctx context.Context, actorID, slotID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	ListPending(ctx context.Context, actorID, slotID string) ([]PendingRequest, error)
	Approve(ctx context.Context, actorID, slotID, requesterID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Reject(ctx context.Context, actorID, slotID, requesterID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Start(ctx context.Context, actorID, slotID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
	Complete(ctx context.Context, actorID, slotID, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
}
