package blocklist

import (
	"context"
	"errors"
	"strings"
	"time"
)

var ErrInvalidTarget = errors.New("invalid block target")

type UserSummary struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"displayName"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
}

type Store interface {
	Block(ctx context.Context, blockerID, blockedID string, now time.Time) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	List(ctx context.Context, blockerID string, limit int) ([]UserSummary, error)
}

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) (*Service, error) {
	if store == nil { return nil, errors.New("block store is required") }
	return &Service{store: store, now: time.Now}, nil
}

func (s *Service) Block(ctx context.Context, blockerID, blockedID string) error {
	blockerID = strings.TrimSpace(blockerID)
	blockedID = strings.TrimSpace(blockedID)
	if blockerID == "" || blockedID == "" || blockerID == blockedID { return ErrInvalidTarget }
	return s.store.Block(ctx, blockerID, blockedID, s.now().UTC())
}

func (s *Service) Unblock(ctx context.Context, blockerID, blockedID string) error {
	blockerID = strings.TrimSpace(blockerID)
	blockedID = strings.TrimSpace(blockedID)
	if blockerID == "" || blockedID == "" || blockerID == blockedID { return ErrInvalidTarget }
	return s.store.Unblock(ctx, blockerID, blockedID)
}

func (s *Service) List(ctx context.Context, blockerID string) ([]UserSummary, error) {
	blockerID = strings.TrimSpace(blockerID)
	if blockerID == "" { return nil, ErrInvalidTarget }
	return s.store.List(ctx, blockerID, 200)
}
