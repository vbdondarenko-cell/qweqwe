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

type submitRewardedViewRequest struct {
	Receipt string `json:"receipt"`
}

type verifyPurchaseRequest struct {
	PurchaseToken string `json:"purchaseToken"`
}

// submitRewardedView and verifyPurchase never trust the request body's own
// claims about outcome (there are none to trust -- the raw receipt/token is
// opaque to this handler) — Service.SubmitRewardedView/VerifyPurchase
// always re-verify with the configured provider adapter before anything is
// recorded. See docs/LINKUP_PLUS_MONETIZATION.md §6/§8's explicit ban on
// client-authoritative watched=true/purchase state.
func (s *Server) submitRewardedView(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Monetization == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization service is unavailable")
		return
	}
	var in submitRewardedViewRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid rewarded-view request")
		return
	}
	snapshot, err := s.deps.Monetization.SubmitRewardedView(r.Context(), auth.User.ID, in.Receipt)
	if err != nil {
		s.writeMonetizationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) verifyPurchase(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Monetization == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization service is unavailable")
		return
	}
	var in verifyPurchaseRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid purchase request")
		return
	}
	snapshot, err := s.deps.Monetization.VerifyPurchase(r.Context(), auth.User.ID, in.PurchaseToken)
	if err != nil {
		s.writeMonetizationError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, snapshot)
}

func (s *Server) writeMonetizationError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, monetization.ErrCapabilityDisabled):
		writeProblem(w, r, http.StatusServiceUnavailable, "capability_disabled", "this monetization capability is not configured yet")
	case errors.Is(err, monetization.ErrRewardedViewRejected):
		writeProblem(w, r, http.StatusUnprocessableEntity, "rewarded_view_rejected", "the rewarded ad view could not be verified")
	case errors.Is(err, monetization.ErrRewardedTooSoon):
		writeProblem(w, r, http.StatusConflict, "rewarded_view_too_soon", "wait for the minimum interval between rewarded views")
	case errors.Is(err, monetization.ErrRewardedCooldownActive):
		writeProblem(w, r, http.StatusConflict, "rewarded_cooldown_active", "the free-premium claim cooldown is still active")
	case errors.Is(err, monetization.ErrPurchaseRejected):
		writeProblem(w, r, http.StatusUnprocessableEntity, "purchase_rejected", "the purchase could not be verified")
	case errors.Is(err, monetization.ErrInvalidPurchase):
		writeProblem(w, r, http.StatusUnprocessableEntity, "invalid_purchase", "the verified purchase response was malformed")
	default:
		writeProblem(w, r, http.StatusServiceUnavailable, "monetization_not_ready", "monetization state is temporarily unavailable")
	}
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
