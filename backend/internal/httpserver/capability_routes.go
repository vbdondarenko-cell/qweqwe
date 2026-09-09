package httpserver

import (
	"log/slog"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
)

func (s *Server) capabilities(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Capabilities == nil {
		writeJSON(w, http.StatusOK, capability.DisabledSnapshot())
		return
	}
	snapshot, err := s.deps.Capabilities.Snapshot(r.Context(), auth.User.ID)
	if err != nil {
		slog.Warn("capability registry read failed; returning fail-closed snapshot", "user_id", auth.User.ID, "error", err)
		writeJSON(w, http.StatusOK, capability.DisabledSnapshot())
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) requireCapability(key capability.Key, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth, ok := authFrom(r)
		if !ok {
			writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
			return
		}
		if s.deps.Capabilities == nil || !s.deps.Capabilities.Enabled(r.Context(), auth.User.ID, key) {
			writeProblem(w, r, http.StatusForbidden, "capability_disabled", "capability is not active")
			return
		}
		next.ServeHTTP(w, r)
	})
}
