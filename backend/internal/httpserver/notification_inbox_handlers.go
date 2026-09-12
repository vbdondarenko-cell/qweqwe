package httpserver

import (
	"net/http"
	"strconv"
)

// listNotifications and markNotificationsRead expose the real, persisted
// in-app notification history (notification_deliveries, unchanged since
// README §6.8 / migration 000027 -- only read_at, migration 000040, is
// new) behind GET/POST /v1/me/notifications. Previously nothing ever read
// this table back for a client; the bell icon the frozen design reference
// specifies had no real data source at all.
func (s *Server) listNotifications(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.NotificationInbox == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "notification inbox is unavailable")
		return
	}
	limit := 0
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if parsed, err := strconv.Atoi(raw); err == nil {
			limit = parsed
		}
	}
	var before int64
	if raw := r.URL.Query().Get("before"); raw != "" {
		if parsed, err := strconv.ParseInt(raw, 10, 64); err == nil {
			before = parsed
		}
	}
	snapshot, err := s.deps.NotificationInbox.List(r.Context(), auth.User.ID, limit, before)
	if err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) markNotificationsRead(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.NotificationInbox == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "notification inbox is unavailable")
		return
	}
	if err := s.deps.NotificationInbox.MarkAllRead(r.Context(), auth.User.ID); err != nil {
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
