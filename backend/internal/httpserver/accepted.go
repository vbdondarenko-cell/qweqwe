package httpserver

import (
	"net/http"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

func (s *Server) listAccepted(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	items, err := s.deps.Slots.ListAccepted(r.Context(), auth.User.ID, r.PathValue("slotID"))
	if err != nil { s.writeSlotError(w, r, err); return }
	writeJSON(w, http.StatusOK, struct { Items []slot.Organizer `json:"items"` }{Items: items})
}
