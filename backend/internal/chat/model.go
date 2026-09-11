package chat

import (
	"context"
	"errors"
	"time"
)

const (
	MaxMessageRunes      = 2000
	MaxRecent            = 100
	MinIdempotencyKeyLen = 16
	MaxIdempotencyKeyLen = 128
)

var (
	ErrInvalidInput        = errors.New("invalid chat input")
	ErrNotFound            = errors.New("chat slot not found")
	ErrForbidden           = errors.New("chat forbidden")
	ErrClosed              = errors.New("chat closed")
	ErrIdempotencyConflict = errors.New("chat idempotency conflict")
)

type Author struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
}

// MessageKind distinguishes a user-authored chat message from a
// server-originated system notice about a Slot lifecycle event. USER is the
// v1.0 baseline shape; SYSTEM is additive (README §6.7 Chat V2).
type MessageKind string

const (
	KindUser   MessageKind = "USER"
	KindSystem MessageKind = "SYSTEM"
)

// SystemEventType enumerates the Slot lifecycle events a SYSTEM message can
// report. The server never composes localized free text for these: it emits
// the typed event plus an optional Subject user so the client renders its
// own localized string. New values are additive in future migrations/blocks.
type SystemEventType string

const (
	SystemEventSlotStarted SystemEventType = "SLOT_STARTED"
)

type Message struct {
	ID     string      `json:"id"`
	SlotID string      `json:"slotId"`
	Kind   MessageKind `json:"kind"`
	// Author is present for Kind==USER and nil for Kind==SYSTEM: nobody
	// "said" a system notice.
	Author *Author `json:"author,omitempty"`
	// Text carries the user-authored body for Kind==USER; empty for SYSTEM.
	Text string `json:"text,omitempty"`
	// SystemEventType and Subject are present only for Kind==SYSTEM.
	SystemEventType *SystemEventType `json:"systemEventType,omitempty"`
	Subject         *Author          `json:"subject,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
}

type Store interface {
	Send(ctx context.Context, actorID, slotID, messageID, idempotencyKey, text string) (Message, error)
	ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]Message, error)
}
