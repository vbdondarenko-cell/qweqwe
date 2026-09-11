package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/notification"
)

func (s *Server) getNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.NotificationPreferences == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "notification preferences service is unavailable")
		return
	}
	prefs, err := s.deps.NotificationPreferences.Get(r.Context(), auth.User.ID)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (s *Server) updateNotificationPreferences(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.NotificationPreferences == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "notification preferences service is unavailable")
		return
	}
	var in notification.StoredPreferences
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	saved, err := s.deps.NotificationPreferences.Update(r.Context(), auth.User.ID, in)
	if err != nil {
		if errors.Is(err, notification.ErrInvalidPreferences) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_preferences", "invalid notification preferences")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, saved)
}
