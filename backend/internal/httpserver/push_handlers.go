package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
)

type registerAndroidPushRequest struct {
	InstallationID string `json:"installationId"`
	Token          string `json:"token"`
	AppVersion     string `json:"appVersion,omitempty"`
}

func (s *Server) registerAndroidPush(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Push == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "push service is unavailable")
		return
	}
	var in registerAndroidPushRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	if err := s.deps.Push.RegisterAndroid(r.Context(), auth.User.ID, auth.SessionID, in.InstallationID, in.Token, in.AppVersion); err != nil {
		if errors.Is(err, push.ErrInvalidRegistration) || errors.Is(err, push.ErrInvalidInstallation) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_push_registration", "invalid push registration")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) revokeAndroidPush(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Push == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "push service is unavailable")
		return
	}
	if err := s.deps.Push.RevokeInstallation(r.Context(), auth.User.ID, r.PathValue("installationID")); err != nil {
		if errors.Is(err, push.ErrInvalidInstallation) {
			writeProblem(w, r, http.StatusBadRequest, "invalid_installation_id", "invalid installation id")
			return
		}
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
