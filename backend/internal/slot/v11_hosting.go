package slot

import (
	"context"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
)

// V11HostingStore is an additive capability implemented by the v1.1 store.
// Keeping it separate preserves the stable v1.0 Store contract while hosting
// draft/publish is brought online behind explicit API surfaces.
type V11HostingStore interface {
	CreateDraft(ctx context.Context, actorID string, candidate Slot, idempotencyKey string, requestHash []byte) (Slot, error)
	PublishDraft(ctx context.Context, actorID, slotID string, expectedVersion int64, idempotencyKey string, requestHash []byte, now time.Time) (Slot, error)
}

func (s *Service) CreateDraft(ctx context.Context, actorID string, in CreateInput, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	key, ok := validIdempotencyKey(idempotencyKey)
	store, supported := s.store.(V11HostingStore)
	if actorID == "" || !ok || !supported {
		return Slot{}, ErrInvalidInput
	}
	if err := normalizeCreate(&in); err != nil {
		return Slot{}, err
	}
	id, err := identifier.NewUUID()
	if err != nil {
		return Slot{}, err
	}
	now := s.now().UTC()
	candidate := Slot{
		ID:               id,
		Organizer:        Organizer{ID: actorID},
		Title:            in.Title,
		Activity:         in.Activity,
		Details:          in.Details,
		PlaceText:        in.PlaceText,
		ZoneText:         in.ZoneText,
		CanonicalPlaceID: in.CanonicalPlaceID,
		StartAt:          in.StartAt,
		Capacity:         in.Capacity,
		AcceptedCount:    0,
		State:            StateDraft,
		AccessMode:       effectiveAccessMode(in.AccessMode),
		Visibility:       effectiveVisibility(in.Visibility),
		ViewerState:      ViewerHost,
		Version:          1,
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	fingerprint, err := hashRequest(struct {
		Title            string
		Activity         string
		Details          *string
		PlaceText        string
		ZoneText         *string
		CanonicalPlaceID *string
		StartAt          *time.Time
		Capacity         int
		AccessMode       AccessMode
		Visibility       Visibility
		Draft            bool
	}{in.Title, in.Activity, in.Details, in.PlaceText, in.ZoneText, in.CanonicalPlaceID, in.StartAt, in.Capacity, effectiveAccessMode(in.AccessMode), effectiveVisibility(in.Visibility), true})
	if err != nil {
		return Slot{}, err
	}
	return store.CreateDraft(ctx, actorID, candidate, key, fingerprint)
}

func (s *Service) PublishDraft(ctx context.Context, actorID, slotID string, expectedVersion int64, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	key, ok := validIdempotencyKey(idempotencyKey)
	store, supported := s.store.(V11HostingStore)
	if actorID == "" || slotID == "" || expectedVersion < 1 || !ok || !supported {
		return Slot{}, ErrInvalidInput
	}
	fingerprint, err := hashRequest(struct {
		SlotID          string
		ExpectedVersion int64
		Publish         bool
	}{slotID, expectedVersion, true})
	if err != nil {
		return Slot{}, err
	}
	return store.PublishDraft(ctx, actorID, slotID, expectedVersion, key, fingerprint, s.now().UTC())
}
