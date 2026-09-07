package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/monetization"
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

type bindReferralRequest struct {
	Code string `json:"code"`
}

func (s *Server) bindReferral(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Monetization == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization service is unavailable")
		return
	}
	var input bindReferralRequest
	if err := decodeJSON(w, r, &input); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid referral request")
		return
	}
	if err := s.deps.Monetization.BindReferral(r.Context(), auth.User.ID, input.Code); err != nil {
		switch {
		case errors.Is(err, monetization.ErrInvalidReferralCode):
			writeProblem(w, r, http.StatusBadRequest, "invalid_referral_code", "referral code is invalid")
		case errors.Is(err, monetization.ErrSelfReferral):
			writeProblem(w, r, http.StatusConflict, "self_referral", "you cannot use your own referral code")
		case errors.Is(err, monetization.ErrReferralAlreadyBound):
			writeProblem(w, r, http.StatusConflict, "referral_already_bound", "a referral code is already bound to this account")
		case errors.Is(err, monetization.ErrReferralExpired):
			writeProblem(w, r, http.StatusConflict, "referral_expired", "the referral qualification window has expired")
		default:
			writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "referral service is temporarily unavailable")
		}
		return
	}
	snapshot, err := s.deps.Monetization.Snapshot(r.Context(), auth.User.ID)
	if err != nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization state is temporarily unavailable")
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}
