package slot

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"sort"
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
	// Same reasoning for visibility: v1.0 mandates Public (README §4.3).
	// PRIVATE is configured through DRAFT, same as INSTANT/WAITLIST above.
	if in.Visibility != nil && *in.Visibility != VisibilityPublic {
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

// ListPulse's sort defaults to PulseSortRecency when rawSort is empty —
// the long-standing, unchanged behavior — and accepts "RELEVANCE"
// (case-insensitive) to opt into README §6.10's Layer 2 ranking. Any
// other value is a clear input error, not a silent fallback to recency.
func (s *Service) ListPulse(ctx context.Context, actorID, rawSort string) ([]Slot, error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return nil, ErrInvalidInput
	}
	sort := PulseSortRecency
	if trimmed := strings.ToUpper(strings.TrimSpace(rawSort)); trimmed != "" {
		switch PulseSort(trimmed) {
		case PulseSortRecency, PulseSortRelevance:
			sort = PulseSort(trimmed)
		default:
			return nil, ErrInvalidInput
		}
	}
	return s.store.ListPulse(ctx, actorID, 100, sort)
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
	if in.Visibility != nil {
		visibility := Visibility(strings.ToUpper(strings.TrimSpace(string(*in.Visibility))))
		if !validVisibility(visibility) {
			return ErrInvalidInput
		}
		in.Visibility = &visibility
	}
	switch effectiveVisibility(in.Visibility) {
	case VisibilitySelected:
		ids, err := normalizeSelectedUserIDs(in.SelectedUserIDs)
		if err != nil {
			return err
		}
		in.SelectedUserIDs = ids
		in.LassoPolygonWKT, in.CorridorLineWKT, in.CorridorRadiusM = nil, nil, nil
	case VisibilityLasso:
		wkt, err := normalizeLassoPolygon(in.LassoPolygonWKT)
		if err != nil {
			return err
		}
		in.LassoPolygonWKT = &wkt
		in.SelectedUserIDs, in.CorridorLineWKT, in.CorridorRadiusM = nil, nil, nil
	case VisibilityTravelCorridor:
		line, radius, err := normalizeCorridor(in.CorridorLineWKT, in.CorridorRadiusM)
		if err != nil {
			return err
		}
		in.CorridorLineWKT, in.CorridorRadiusM = &line, &radius
		in.SelectedUserIDs, in.LassoPolygonWKT = nil, nil
	default:
		// Ignore any accidentally-provided geometry/allow-list rather than
		// silently storing it against a Slot whose visibility never
		// actually reads it.
		in.SelectedUserIDs, in.LassoPolygonWKT, in.CorridorLineWKT, in.CorridorRadiusM = nil, nil, nil, nil
	}
	return nil
}

// maxGeometryWKTLength mirrors migration 000035's own
// slots_lasso_polygon_wkt_length/slots_corridor_line_wkt_length CHECK
// constraints — validated here too so a too-large payload is a clean
// ErrInvalidInput at the domain boundary, not a database constraint
// violation surfacing as an opaque 500.
const maxGeometryWKTLength = 200000

// normalizeLassoPolygon requires a plausible WKT POLYGON: non-empty,
// within the same length bound migration 000035 enforces, and starting
// with the POLYGON keyword (case-insensitive, ignoring leading
// whitespace). Real geometry validity (closed ring, no self-intersection,
// etc.) is PostGIS's job at query time via st_geomfromtext, not
// reimplemented here — this is a shape-of-input check, not a geometry
// parser.
func normalizeLassoPolygon(raw *string) (string, error) {
	if raw == nil {
		return "", ErrInvalidInput
	}
	wkt := strings.TrimSpace(*raw)
	if len(wkt) < 10 || len(wkt) > maxGeometryWKTLength || !strings.HasPrefix(strings.ToUpper(wkt), "POLYGON") {
		return "", ErrInvalidInput
	}
	return wkt, nil
}

// normalizeCorridor requires both a plausible WKT LINESTRING and a radius
// in the same 1..50000 meter range migration 000035 enforces — together
// or not at all (an EditInput/CreateInput carrying only one of the two is
// invalid input, not a silently-ignored partial configuration).
func normalizeCorridor(rawLine *string, rawRadius *int) (string, int, error) {
	if rawLine == nil || rawRadius == nil {
		return "", 0, ErrInvalidInput
	}
	line := strings.TrimSpace(*rawLine)
	if len(line) < 10 || len(line) > maxGeometryWKTLength || !strings.HasPrefix(strings.ToUpper(line), "LINESTRING") {
		return "", 0, ErrInvalidInput
	}
	if *rawRadius < 1 || *rawRadius > 50000 {
		return "", 0, ErrInvalidInput
	}
	return line, *rawRadius, nil
}

// normalizeSelectedUserIDs validates the VisibilitySelected allow-list: at
// least one entry, each a well-formed UUID, deduplicated, capped at
// MaxPendingRequests (reusing the same limit README gives no specific
// number for, but which already exists as a reasonable per-Slot people-list
// ceiling elsewhere in this package).
func normalizeSelectedUserIDs(raw []string) ([]string, error) {
	if len(raw) == 0 {
		return nil, ErrInvalidInput
	}
	seen := make(map[string]bool, len(raw))
	out := make([]string, 0, len(raw))
	for _, id := range raw {
		v := strings.ToLower(strings.TrimSpace(id))
		if !validUUID(v) {
			return nil, ErrInvalidInput
		}
		if seen[v] {
			continue
		}
		seen[v] = true
		out = append(out, v)
	}
	if len(out) > MaxPendingRequests {
		return nil, ErrInvalidInput
	}
	// Sorted so the idempotency fingerprint (hashRequest) is stable
	// regardless of the order the caller listed the same set of IDs in.
	sort.Strings(out)
	return out, nil
}

func normalizeEdit(in *EditInput) error {
	if in.Title == nil && in.Details == nil && in.PlaceText == nil && in.ZoneText == nil && in.CanonicalPlaceID == nil && !in.ClearCanonicalPlaceID && in.StartAt == nil && !in.ClearStartAt && in.Capacity == nil && in.AccessMode == nil && in.Visibility == nil {
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
	if in.Visibility != nil {
		visibility := Visibility(strings.ToUpper(strings.TrimSpace(string(*in.Visibility))))
		if !validVisibility(visibility) {
			return ErrInvalidInput
		}
		switch visibility {
		case VisibilitySelected:
			ids, err := normalizeSelectedUserIDs(in.SelectedUserIDs)
			if err != nil {
				return err
			}
			in.SelectedUserIDs = ids
		case VisibilityLasso, VisibilityTravelCorridor:
			// EditInput has no field to configure a Lasso polygon or
			// Travel Corridor route/radius — accepting either here would
			// let a host switch a Slot to one of these modes with no
			// shape at all, silently undiscoverable rather than a clear
			// error. Matches SELECTED's own original first-block scope
			// before EditInput.SelectedUserIDs existed.
			return ErrInvalidInput
		default:
			// Ignore any accidentally-provided list rather than storing it
			// against a Slot whose visibility this edit is not setting to
			// SELECTED at all.
			in.SelectedUserIDs = nil
		}
		in.Visibility = &visibility
	} else {
		// No visibility change requested at all — the allow-list is never
		// touched implicitly (SelectedUserIDs.doc comment).
		in.SelectedUserIDs = nil
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

func effectiveVisibility(visibility *Visibility) Visibility {
	if visibility == nil {
		return VisibilityPublic
	}
	return *visibility
}

// validVisibility is the closed set this API accepts: every README §4.3
// mode now has a real implementation — see VisibilityPrivate's,
// VisibilityLinks's, VisibilitySelected's, VisibilityCity's,
// VisibilityLasso's and VisibilityTravelCorridor's own doc comments for
// each mode's specific scope/limits.
func validVisibility(visibility Visibility) bool {
	return visibility == VisibilityPublic || visibility == VisibilityPrivate ||
		visibility == VisibilityLinks || visibility == VisibilitySelected ||
		visibility == VisibilityCity || visibility == VisibilityLasso ||
		visibility == VisibilityTravelCorridor
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
