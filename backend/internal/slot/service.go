package slot

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/identifier"
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

func (s *Service) Create(ctx context.Context, actorID string, in CreateInput, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	key, ok := validIdempotencyKey(idempotencyKey)
	if actorID == "" || !ok {
		return Slot{}, ErrInvalidInput
	}
	if err := normalizeCreate(&in); err != nil {
		return Slot{}, err
	}
	// The legacy immediate-create route remains the stable v1.0 APPROVAL flow.
	// INSTANT/WAITLIST are configured through DRAFT and only publish after their
	// dedicated concurrency semantics are available.
	if in.AccessMode != nil && *in.AccessMode != AccessApproval {
		return Slot{}, ErrInvalidState
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
		State:            StateFilling,
		AccessMode:       effectiveAccessMode(in.AccessMode),
		Visibility:       VisibilityPublic,
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
	}{in.Title, in.Activity, in.Details, in.PlaceText, in.ZoneText, in.CanonicalPlaceID, in.StartAt, in.Capacity, effectiveAccessMode(in.AccessMode)})
	if err != nil {
		return Slot{}, err
	}
	return s.store.Create(ctx, actorID, candidate, key, fingerprint)
}

func (s *Service) Get(ctx context.Context, actorID, slotID string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" {
		return Slot{}, ErrInvalidInput
	}
	return s.store.Get(ctx, actorID, slotID)
}

func (s *Service) ListPulse(ctx context.Context, actorID string) ([]Slot, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalidInput
	}
	return s.store.ListPulse(ctx, actorID, 100)
}

func (s *Service) Edit(ctx context.Context, actorID, slotID string, patch EditInput, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	key, ok := validIdempotencyKey(idempotencyKey)
	if actorID == "" || slotID == "" || patch.ExpectedVersion < 1 || !ok {
		return Slot{}, ErrInvalidInput
	}
	if err := normalizeEdit(&patch); err != nil {
		return Slot{}, err
	}
	fingerprint, err := hashRequest(struct {
		SlotID string
		Patch  EditInput
	}{slotID, patch})
	if err != nil {
		return Slot{}, err
	}
	return s.store.Edit(ctx, actorID, slotID, patch, key, fingerprint, s.now().UTC())
}

func (s *Service) Cancel(ctx context.Context, actorID, slotID string, expectedVersion int64, idempotencyKey string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	key, ok := validIdempotencyKey(idempotencyKey)
	if actorID == "" || slotID == "" || expectedVersion < 1 || !ok {
		return Slot{}, ErrInvalidInput
	}
	fingerprint, err := hashRequest(struct {
		SlotID          string
		ExpectedVersion int64
	}{slotID, expectedVersion})
	if err != nil {
		return Slot{}, err
	}
	return s.store.Cancel(ctx, actorID, slotID, expectedVersion, key, fingerprint, s.now().UTC())
}

func (s *Service) Request(ctx context.Context, actorID, slotID, idempotencyKey string) (Slot, error) {
	actorID, slotID, key, err := normalizeMutation(actorID, slotID, idempotencyKey)
	if err != nil {
		return Slot{}, err
	}
	fingerprint, err := hashRequest(struct{ SlotID string }{slotID})
	if err != nil {
		return Slot{}, err
	}
	return s.store.Request(ctx, actorID, slotID, key, fingerprint, s.now().UTC())
}

func (s *Service) Leave(ctx context.Context, actorID, slotID, idempotencyKey string) (Slot, error) {
	actorID, slotID, key, err := normalizeMutation(actorID, slotID, idempotencyKey)
	if err != nil {
		return Slot{}, err
	}
	fingerprint, err := hashRequest(struct{ SlotID string }{slotID})
	if err != nil {
		return Slot{}, err
	}
	return s.store.Leave(ctx, actorID, slotID, key, fingerprint, s.now().UTC())
}

func (s *Service) ListPending(ctx context.Context, actorID, slotID string) ([]PendingRequest, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	if actorID == "" || slotID == "" {
		return nil, ErrInvalidInput
	}
	return s.store.ListPending(ctx, actorID, slotID)
}

func (s *Service) Approve(ctx context.Context, actorID, slotID, requesterID, idempotencyKey string) (Slot, error) {
	return s.hostRequestMutation(ctx, actorID, slotID, requesterID, idempotencyKey, "approve")
}

func (s *Service) Reject(ctx context.Context, actorID, slotID, requesterID, idempotencyKey string) (Slot, error) {
	return s.hostRequestMutation(ctx, actorID, slotID, requesterID, idempotencyKey, "reject")
}

func (s *Service) hostRequestMutation(ctx context.Context, actorID, slotID, requesterID, idempotencyKey, operation string) (Slot, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	requesterID = strings.TrimSpace(requesterID)
	key, ok := validIdempotencyKey(idempotencyKey)
	if actorID == "" || slotID == "" || requesterID == "" || actorID == requesterID || !ok {
		return Slot{}, ErrInvalidInput
	}
	fingerprint, err := hashRequest(struct {
		SlotID      string
		RequesterID string
	}{slotID, requesterID})
	if err != nil {
		return Slot{}, err
	}
	now := s.now().UTC()
	if operation == "approve" {
		return s.store.Approve(ctx, actorID, slotID, requesterID, key, fingerprint, now)
	}
	return s.store.Reject(ctx, actorID, slotID, requesterID, key, fingerprint, now)
}

func (s *Service) Start(ctx context.Context, actorID, slotID, idempotencyKey string) (Slot, error) {
	return s.lifecycleMutation(ctx, actorID, slotID, idempotencyKey, "start")
}

func (s *Service) Complete(ctx context.Context, actorID, slotID, idempotencyKey string) (Slot, error) {
	return s.lifecycleMutation(ctx, actorID, slotID, idempotencyKey, "complete")
}

func (s *Service) lifecycleMutation(ctx context.Context, actorID, slotID, idempotencyKey, operation string) (Slot, error) {
	actorID, slotID, key, err := normalizeMutation(actorID, slotID, idempotencyKey)
	if err != nil {
		return Slot{}, err
	}
	fingerprint, err := hashRequest(struct{ SlotID string }{slotID})
	if err != nil {
		return Slot{}, err
	}
	now := s.now().UTC()
	if operation == "start" {
		return s.store.Start(ctx, actorID, slotID, key, fingerprint, now)
	}
	return s.store.Complete(ctx, actorID, slotID, key, fingerprint, now)
}

func normalizeMutation(actorID, slotID, idempotencyKey string) (string, string, string, error) {
	actorID = strings.TrimSpace(actorID)
	slotID = strings.TrimSpace(slotID)
	key, ok := validIdempotencyKey(idempotencyKey)
	if actorID == "" || slotID == "" || !ok {
		return "", "", "", ErrInvalidInput
	}
	return actorID, slotID, key, nil
}

func normalizeCreate(in *CreateInput) error {
	in.Title = strings.TrimSpace(in.Title)
	in.Activity = strings.ToLower(strings.TrimSpace(in.Activity))
	in.PlaceText = strings.TrimSpace(in.PlaceText)
	if len([]rune(in.Title)) < 1 || len([]rune(in.Title)) > 120 || len([]rune(in.Activity)) < 1 || len([]rune(in.Activity)) > 64 || len([]rune(in.PlaceText)) < 1 || len([]rune(in.PlaceText)) > 240 || in.Capacity < 2 || in.Capacity > MaxV1Capacity {
		return ErrInvalidInput
	}
	if in.Details != nil {
		v := strings.TrimSpace(*in.Details)
		if len([]rune(v)) > 2000 {
			return ErrInvalidInput
		}
		if v == "" {
			in.Details = nil
		} else {
			in.Details = &v
		}
	}
	if in.ZoneText != nil {
		v := strings.TrimSpace(*in.ZoneText)
		if len([]rune(v)) > 160 {
			return ErrInvalidInput
		}
		if v == "" {
			in.ZoneText = nil
		} else {
			in.ZoneText = &v
		}
	}
	if in.CanonicalPlaceID != nil {
		v := strings.ToLower(strings.TrimSpace(*in.CanonicalPlaceID))
		if !validUUID(v) {
			return ErrInvalidInput
		}
		in.CanonicalPlaceID = &v
	}
	if in.StartAt != nil {
		v := in.StartAt.UTC()
		in.StartAt = &v
	}
	if in.AccessMode != nil {
		mode := AccessMode(strings.ToUpper(strings.TrimSpace(string(*in.AccessMode))))
		if !validAccessMode(mode) {
			return ErrInvalidInput
		}
		in.AccessMode = &mode
	}
	return nil
}

func normalizeEdit(in *EditInput) error {
	if in.Title == nil && in.Details == nil && in.PlaceText == nil && in.ZoneText == nil && in.CanonicalPlaceID == nil && !in.ClearCanonicalPlaceID && in.StartAt == nil && !in.ClearStartAt && in.Capacity == nil && in.AccessMode == nil {
		return ErrInvalidInput
	}
	if (in.StartAt != nil && in.ClearStartAt) || (in.CanonicalPlaceID != nil && in.ClearCanonicalPlaceID) {
		return ErrInvalidInput
	}
	if in.Title != nil {
		v := strings.TrimSpace(*in.Title)
		if len([]rune(v)) < 1 || len([]rune(v)) > 120 {
			return ErrInvalidInput
		}
		in.Title = &v
	}
	if in.Details != nil {
		v := strings.TrimSpace(*in.Details)
		if len([]rune(v)) > 2000 {
			return ErrInvalidInput
		}
		in.Details = &v
	}
	if in.PlaceText != nil {
		v := strings.TrimSpace(*in.PlaceText)
		if len([]rune(v)) < 1 || len([]rune(v)) > 240 {
			return ErrInvalidInput
		}
		in.PlaceText = &v
	}
	if in.ZoneText != nil {
		v := strings.TrimSpace(*in.ZoneText)
		if len([]rune(v)) > 160 {
			return ErrInvalidInput
		}
		in.ZoneText = &v
	}
	if in.CanonicalPlaceID != nil {
		v := strings.ToLower(strings.TrimSpace(*in.CanonicalPlaceID))
		if !validUUID(v) {
			return ErrInvalidInput
		}
		in.CanonicalPlaceID = &v
	}
	if in.StartAt != nil {
		v := in.StartAt.UTC()
		in.StartAt = &v
	}
	if in.Capacity != nil && (*in.Capacity < 2 || *in.Capacity > MaxV1Capacity) {
		return ErrInvalidInput
	}
	if in.AccessMode != nil {
		mode := AccessMode(strings.ToUpper(strings.TrimSpace(string(*in.AccessMode))))
		if !validAccessMode(mode) {
			return ErrInvalidInput
		}
		in.AccessMode = &mode
	}
	return nil
}

func effectiveAccessMode(mode *AccessMode) AccessMode {
	if mode == nil {
		return AccessApproval
	}
	return *mode
}

func validAccessMode(mode AccessMode) bool {
	return mode == AccessInstant || mode == AccessApproval || mode == AccessWaitlist
}

func validUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for i := 0; i < len(value); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		c := value[i]
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return false
		}
	}
	return true
}

func validIdempotencyKey(raw string) (string, bool) {
	key := strings.TrimSpace(raw)
	if len(key) < 16 || len(key) > 128 {
		return "", false
	}
	for _, r := range key {
		if r < 33 || r > 126 {
			return "", false
		}
	}
	return key, true
}

func hashRequest(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(raw)
	return sum[:], nil
}
