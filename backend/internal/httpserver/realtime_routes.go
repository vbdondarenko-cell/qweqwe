package httpserver

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/realtime"
)

type RealtimeFeed interface {
	Pull(context.Context, string, int64, int) (realtime.ViewerBatch, error)
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
