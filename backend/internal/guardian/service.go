package guardian

import (
	"context"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

type Service struct {
	store Store
	now   func() time.Time
}

func NewService(store Store) (*Service, error) {
	if store == nil {
		return nil, ErrInvalidInput
	}
	return &Service{store: store, now: time.Now}, nil
}

// CreateLink issues a new opaque guardian link. See this package's own doc
// comment for why every Mode except ModeStatusOnly is rejected outright.
func (s *Service) CreateLink(ctx context.Context, actorID, slotID string, mode Mode, ttl time.Duration) (LinkView, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" {
		return LinkView{}, ErrInvalidInput
	}
	if mode != ModeStatusOnly {
		return LinkView{}, ErrModeNotSupported
	}
	if ttl < MinTTL || ttl > MaxTTL {
		return LinkView{}, ErrInvalidInput
	}
	id, err := identifier.NewUUID()
	if err != nil {
		return LinkView{}, err
	}
	raw, digest, err := session.Generate()
	if err != nil {
		return LinkView{}, err
	}
	now := s.now().UTC()
	expiresAt := now.Add(ttl)
	if err := s.store.CreateLink(ctx, id, slotID, actorID, mode, digest[:], now, expiresAt); err != nil {
		return LinkView{}, err
	}
	return LinkView{
		Link:  Link{ID: id, SlotID: slotID, CreatedBy: actorID, Mode: mode, CreatedAt: now, ExpiresAt: expiresAt},
		Token: raw,
	}, nil
}

// Revoke burns a link early. Only its own creator can revoke it.
func (s *Service) Revoke(ctx context.Context, actorID, linkID string) error {
	actorID = strings.TrimSpace(actorID)
	linkID = strings.TrimSpace(linkID)
	if actorID == "" || linkID == "" {
		return ErrInvalidInput
	}
	return s.store.Revoke(ctx, linkID, actorID, s.now().UTC())
}

// Access resolves a raw bearer token (never authenticated — the holder may
// not have a LinkUp account at all) to a Status.
func (s *Service) Access(ctx context.Context, rawToken string) (Status, error) {
	rawToken = strings.TrimSpace(rawToken)
	if rawToken == "" {
		return Status{}, ErrLinkUnavailable
	}
	digest, err := session.Hash(rawToken)
	if err != nil {
		return Status{}, ErrLinkUnavailable
	}
	return s.store.Access(ctx, digest[:], s.now().UTC())
}
