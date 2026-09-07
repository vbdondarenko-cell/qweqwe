package chat

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
)

type Service struct {
	store Store
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &Service{store: store}, nil
}

func (s *Service) Send(ctx context.Context, actorID, slotID, idempotencyKey, text string) (Message, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	text = strings.TrimSpace(text)
	if actorID == "" || slotID == "" || text == "" || !utf8.ValidString(text) || utf8.RuneCountInString(text) > MaxMessageRunes || len(idempotencyKey) < MinIdempotencyKeyLen || len(idempotencyKey) > MaxIdempotencyKeyLen {
		return Message{}, ErrInvalidInput
	}
	messageID, err := identifier.NewUUID()
	if err != nil {
		return Message{}, err
	}
	return s.store.Send(ctx, actorID, slotID, messageID, idempotencyKey, text)
}

func (s *Service) ListRecent(ctx context.Context, actorID, slotID string, limit int) ([]Message, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" || limit < 1 || limit > MaxRecent {
		return nil, ErrInvalidInput
	}
	return s.store.ListRecent(ctx, actorID, slotID, limit)
}
