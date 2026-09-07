package httpserver

import (
	"net/http"
)

func (s *Server) getMonetization(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Monetization == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization service is unavailable")
		return
	}
	snapshot, err := s.deps.Monetization.Snapshot(r.Context(), auth.User.ID)
	if err != nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization state is temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}
