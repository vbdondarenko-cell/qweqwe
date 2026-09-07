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

type Message struct {
	ID             string    `json:"id"`
	SlotID         string    `json:"slotId"`
	Author         Author    `json:"author"`
	Text           string    `json:"text"`
	IdempotencyKey string    `json:"idempotencyKey"`
	CreatedAt      time.Time `json:"createdAt"`
}

type Store interface {
	Send(ctx context.Context, actorID, slotID, messageID, idempotencyKey, text string) (Message, error)
	ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]Message, error)
}
