package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/citycontext"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type RealtimeFeed interface {
	Pull(context.Context, string, int64, int) (realtime.ViewerBatch, error)
	CurrentCursor(context.Context, string) (int64, error)
}

type CityRealtimeFeed interface {
	Pull(context.Context, string, int64, int) (realtime.CityBatch, error)
	CurrentCursor(context.Context, string) (int64, error)
}

type realtimeCursorResponse struct {
	Cursor int64 `json:"cursor"`
}

func (s *Server) realtimeEvents(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Realtime == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "realtime service is unavailable")
		return
	}

	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			writeProblem(w, r, http.StatusBadRequest, "invalid_cursor", "invalid realtime cursor")
			return
		}
		after = parsed
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > realtime.MaxViewerBatch {
			writeProblem(w, r, http.StatusBadRequest, "invalid_limit", "invalid realtime limit")
			return
		}
		limit = parsed
	}

	batch, err := s.deps.Realtime.Pull(r.Context(), auth.User.ID, after, limit)
	if err != nil {
		if errors.Is(err, realtime.ErrInvalidInput) || errors.Is(err, realtime.ErrCursorOutOfOrder) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_realtime_request", "invalid realtime request")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

func (s *Server) realtimeCity(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.CityRealtime == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "city realtime service is unavailable")
		return
	}

	after := int64(0)
	if raw := r.URL.Query().Get("after"); raw != "" {
		parsed, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || parsed < 0 {
			writeProblem(w, r, http.StatusBadRequest, "invalid_cursor", "invalid realtime cursor")
			return
		}
		after = parsed
	}
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > realtime.MaxViewerBatch {
			writeProblem(w, r, http.StatusBadRequest, "invalid_limit", "invalid realtime limit")
			return
		}
		limit = parsed
	}

	batch, err := s.deps.CityRealtime.Pull(r.Context(), auth.User.ID, after, limit)
	if err != nil {
		switch {
		case errors.Is(err, citycontext.ErrNotFound):
			writeProblem(w, r, http.StatusNotFound, "city_context_unavailable", "city context is unavailable")
		case errors.Is(err, realtime.ErrInvalidInput), errors.Is(err, realtime.ErrCursorOutOfOrder):
			writeProblem(w, r, http.StatusBadRequest, "invalid_realtime_request", "invalid realtime request")
		default:
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	writeJSON(w, http.StatusOK, batch)
}

// realtimeCursor is the reconnect/first-run bootstrap endpoint (README
// §6.2's "obtain authoritative snapshot/cursor" step): it returns the
// realtime channel's current position with no events attached, so a
// client with no persisted cursor (fresh install, cleared local storage,
// process-death with lost state) can fast-forward straight to "now"
// instead of pulling forward through the entire outbox history it has no
// use for — its actual current state comes from Pulse/Get/ListMine, which
// are already authoritative on their own.
func (s *Server) realtimeCursor(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Realtime == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "realtime service is unavailable")
		return
	}
	cursor, err := s.deps.Realtime.CurrentCursor(r.Context(), auth.User.ID)
	if err != nil {
		if errors.Is(err, realtime.ErrInvalidInput) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_realtime_request", "invalid realtime request")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, realtimeCursorResponse{Cursor: cursor})
}

// realtimeCityCursor mirrors realtimeCursor for the city channel.
func (s *Server) realtimeCityCursor(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.CityRealtime == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "city realtime service is unavailable")
		return
	}
	cursor, err := s.deps.CityRealtime.CurrentCursor(r.Context(), auth.User.ID)
	if err != nil {
		if errors.Is(err, realtime.ErrInvalidInput) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_realtime_request", "invalid realtime request")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, realtimeCursorResponse{Cursor: cursor})
}
