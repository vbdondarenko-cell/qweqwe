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
	ID               string      `json:"id"`
	Organizer        Organizer   `json:"organizer"`
	Title            string      `json:"title"`
	Activity         string      `json:"activity"`
	Details          *string     `json:"details,omitempty"`
	PlaceText        string      `json:"placeText"`
	ZoneText         *string     `json:"zoneText,omitempty"`
	CanonicalPlaceID *string     `json:"canonicalPlaceId,omitempty"`
	StartAt          *time.Time  `json:"startAt,omitempty"`
	Capacity         int         `json:"capacity"`
	AcceptedCount    int         `json:"acceptedCount"`
	State            State       `json:"state"`
	AccessMode       AccessMode  `json:"accessMode"`
	Visibility       Visibility  `json:"visibility"`
	ViewerState      ViewerState `json:"viewerState"`
	Version          int64       `json:"version"`
	CreatedAt        time.Time   `json:"createdAt"`
	UpdatedAt        time.Time   `json:"updatedAt"`
}

type PendingRequest struct {
	User        Organizer `json:"user"`
	RequestedAt time.Time `json:"requestedAt"`
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
