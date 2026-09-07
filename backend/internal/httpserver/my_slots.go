package httpserver

import "net/http"

func (s *Server) listMySlots(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	view := r.URL.Query().Get("view")
	if view == "" { view = "HOSTING" }
	items, err := s.deps.Slots.ListMine(r.Context(), auth.User.ID, view)
	if err != nil { s.writeSlotError(w, r, err); return }
	writeJSON(w, http.StatusOK, pulseResponse{Items: items})
}
