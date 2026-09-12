package httpserver

import (
	"errors"
	"net/http"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/guardian"
)

type createGuardianLinkRequest struct {
	Mode       string `json:"mode"`
	TTLSeconds int    `json:"ttlSeconds"`
}

type guardianLinkResponse struct {
	ID        string    `json:"id"`
	Token     string    `json:"token"`
	Mode      string    `json:"mode"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (s *Server) createGuardianLink(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Guardians == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "guardian service is unavailable")
		return
	}
	var in createGuardianLinkRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	ttl := time.Duration(in.TTLSeconds) * time.Second
	out, err := s.deps.Guardians.CreateLink(r.Context(), auth.User.ID, r.PathValue("slotID"), guardian.Mode(in.Mode), ttl)
	if err != nil {
		s.writeGuardianError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, guardianLinkResponse{ID: out.ID, Token: out.Token, Mode: string(out.Mode), ExpiresAt: out.ExpiresAt})
}

func (s *Server) revokeGuardianLink(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Guardians == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "guardian service is unavailable")
		return
	}
	if err := s.deps.Guardians.Revoke(r.Context(), auth.User.ID, r.PathValue("linkID")); err != nil {
		s.writeGuardianError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// accessGuardianLink is intentionally unauthenticated: the whole point of
// a Ghost Guardian link is that the holder need not have a LinkUp account.
func (s *Server) accessGuardianLink(w http.ResponseWriter, r *http.Request) {
	if s.deps.Guardians == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "guardian service is unavailable")
		return
	}
	out, err := s.deps.Guardians.Access(r.Context(), r.PathValue("token"))
	if err != nil {
		s.writeGuardianError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) writeGuardianError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, guardian.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid guardian link request")
	case errors.Is(err, guardian.ErrModeNotSupported):
		writeProblem(w, r, http.StatusBadRequest, "mode_not_supported", "this guardian link mode is not supported yet")
	case errors.Is(err, guardian.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "guardian link operation is not allowed")
	case errors.Is(err, guardian.ErrLinkUnavailable):
		writeProblem(w, r, http.StatusNotFound, "guardian_link_unavailable", "this guardian link is unavailable")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
