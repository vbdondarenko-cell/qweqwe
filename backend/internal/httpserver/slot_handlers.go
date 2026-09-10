package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type createSlotRequest struct {
	Title            string           `json:"title"`
	Activity         string           `json:"activity"`
	Details          *string          `json:"details"`
	PlaceText        string           `json:"placeText"`
	ZoneText         *string          `json:"zoneText"`
	CanonicalPlaceID *string          `json:"canonicalPlaceId"`
	StartAt          *time.Time       `json:"startAt"`
	Capacity         int              `json:"capacity"`
	AccessMode       *slot.AccessMode `json:"accessMode"`
}

type editSlotRequest struct {
	ExpectedVersion       int64            `json:"expectedVersion"`
	Title                 *string          `json:"title"`
	Details               *string          `json:"details"`
	PlaceText             *string          `json:"placeText"`
	ZoneText              *string          `json:"zoneText"`
	CanonicalPlaceID      *string          `json:"canonicalPlaceId"`
	ClearCanonicalPlaceID bool             `json:"clearCanonicalPlaceId"`
	StartAt               *time.Time       `json:"startAt"`
	ClearStartAt          bool             `json:"clearStartAt"`
	Capacity              *int             `json:"capacity"`
	AccessMode            *slot.AccessMode `json:"accessMode"`
}

type cancelSlotRequest struct {
	ExpectedVersion int64 `json:"expectedVersion"`
}

type pulseResponse struct {
	Items []slot.Slot `json:"items"`
}

func (s *Server) createSlot(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	var in createSlotRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Slots.Create(r.Context(), auth.User.ID, slot.CreateInput{
		Title:            in.Title,
		Activity:         in.Activity,
		Details:          in.Details,
		PlaceText:        in.PlaceText,
		ZoneText:         in.ZoneText,
		CanonicalPlaceID: in.CanonicalPlaceID,
		StartAt:          in.StartAt,
		Capacity:         in.Capacity,
		AccessMode:       in.AccessMode,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) getSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	out, err := s.deps.Slots.Get(r.Context(), auth.User.ID, r.PathValue("slotID"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listPulse(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	items, err := s.deps.Slots.ListPulse(r.Context(), auth.User.ID)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, pulseResponse{Items: items})
}

func (s *Server) editSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	var in editSlotRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Slots.Edit(r.Context(), auth.User.ID, r.PathValue("slotID"), slot.EditInput{
		ExpectedVersion:       in.ExpectedVersion,
		Title:                 in.Title,
		Details:               in.Details,
		PlaceText:             in.PlaceText,
		ZoneText:              in.ZoneText,
		CanonicalPlaceID:      in.CanonicalPlaceID,
		ClearCanonicalPlaceID: in.ClearCanonicalPlaceID,
		StartAt:               in.StartAt,
		ClearStartAt:          in.ClearStartAt,
		Capacity:              in.Capacity,
		AccessMode:            in.AccessMode,
	}, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) cancelSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	var in cancelSlotRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Slots.Cancel(r.Context(), auth.User.ID, r.PathValue("slotID"), in.ExpectedVersion, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) writeSlotError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, slot.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_slot", "invalid slot data")
	case errors.Is(err, slot.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "operation is not allowed")
	case errors.Is(err, slot.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "slot_not_found", "slot not found")
	case errors.Is(err, slot.ErrRequestNotFound):
		writeProblem(w, r, http.StatusNotFound, "request_not_found", "pending request not found")
	case errors.Is(err, slot.ErrConflict):
		writeProblem(w, r, http.StatusConflict, "slot_version_conflict", "slot version is stale")
	case errors.Is(err, slot.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusConflict, "idempotency_conflict", "idempotency key was already used for a different request")
	case errors.Is(err, slot.ErrInvalidState):
		writeProblem(w, r, http.StatusConflict, "invalid_slot_state", "slot state does not allow this operation")
	case errors.Is(err, slot.ErrCapacityFull):
		writeProblem(w, r, http.StatusConflict, "slot_full", "slot has no available capacity")
	case errors.Is(err, slot.ErrRequestLimit):
		writeProblem(w, r, http.StatusConflict, "request_queue_full", "slot has reached the v1.0 pending request limit")
	case errors.Is(err, slot.ErrDuplicateRequest):
		writeProblem(w, r, http.StatusConflict, "request_exists", "a pending request already exists")
	case errors.Is(err, slot.ErrAlreadyMember):
		writeProblem(w, r, http.StatusConflict, "already_member", "user is already an accepted participant")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
