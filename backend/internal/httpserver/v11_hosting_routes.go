package httpserver

import (
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type publishDraftRequest struct {
	ExpectedVersion int64 `json:"expectedVersion"`
}

func (s *Server) createDraftSlot(w http.ResponseWriter, r *http.Request) {
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
	out, err := s.deps.Slots.CreateDraft(r.Context(), auth.User.ID, slot.CreateInput{
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

func (s *Server) publishDraftSlot(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	var in publishDraftRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if !s.requireWaitlistCapabilityForSlot(w, r, auth.User.ID, r.PathValue("slotID")) {
		return
	}
	out, err := s.deps.Slots.PublishDraft(
		r.Context(),
		auth.User.ID,
		r.PathValue("slotID"),
		in.ExpectedVersion,
		r.Header.Get("Idempotency-Key"),
	)
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
