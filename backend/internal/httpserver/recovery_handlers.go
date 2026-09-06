package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
)

type passwordRecoveryRequest struct {
	Email string `json:"email"`
}

type passwordResetRequest struct {
	Token       string `json:"token"`
	NewPassword string `json:"newPassword"`
}

type acceptedResponse struct {
	Status string `json:"status"`
}

func (s *Server) requestPasswordRecovery(w http.ResponseWriter, r *http.Request) {
	if s.deps.Accounts == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "account service is unavailable")
		return
	}
	var in passwordRecoveryRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := s.deps.Accounts.BeginPasswordReset(r.Context(), in.Email); err != nil {
		switch {
		case errors.Is(err, account.ErrInvalidInput):
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid recovery request")
		case errors.Is(err, account.ErrRecoveryUnavailable):
			writeProblem(w, r, http.StatusServiceUnavailable, "recovery_unavailable", "password recovery is not configured")
		default:
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	// The same response is returned whether or not the email exists.
	writeJSON(w, http.StatusAccepted, acceptedResponse{Status: "accepted"})
}

func (s *Server) resetPassword(w http.ResponseWriter, r *http.Request) {
	if s.deps.Accounts == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "account service is unavailable")
		return
	}
	var in passwordResetRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	if err := s.deps.Accounts.ResetPassword(r.Context(), in.Token, in.NewPassword); err != nil {
		switch {
		case errors.Is(err, account.ErrInvalidInput):
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid password")
		case errors.Is(err, account.ErrUnauthorized):
			writeProblem(w, r, http.StatusBadRequest, "invalid_reset_token", "reset token is invalid or expired")
		default:
			writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
