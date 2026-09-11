package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/bump"
)

type issueBumpChallengeResponse struct {
	Nonce     string `json:"nonce"`
	SlotID    string `json:"slotId"`
	ExpiresAt string `json:"expiresAt"`
}

func (s *Server) issueBumpChallenge(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Bump == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "bump service is unavailable")
		return
	}
	ch, err := s.deps.Bump.IssueChallenge(r.Context(), auth.User.ID, r.PathValue("slotID"))
	if err != nil {
		writeBumpError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, issueBumpChallengeResponse{
		Nonce:     ch.Nonce,
		SlotID:    ch.SlotID,
		ExpiresAt: ch.ExpiresAt.UTC().Format(time.RFC3339),
	})
}

type confirmBumpRequest struct {
	CounterpartUserID string `json:"counterpartUserId"`
	Nonce             string `json:"nonce"`
}

type confirmBumpResponse struct {
	Verified    bool   `json:"verified"`
	ConfirmedAt string `json:"confirmedAt,omitempty"`
}

func (s *Server) confirmBump(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Bump == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "bump service is unavailable")
		return
	}
	var in confirmBumpRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_json", "invalid request body")
		return
	}
	result, err := s.deps.Bump.Confirm(r.Context(), auth.User.ID, r.PathValue("slotID"), in.CounterpartUserID, in.Nonce)
	if err != nil {
		writeBumpError(w, r, err)
		return
	}
	resp := confirmBumpResponse{Verified: result.Verified}
	if result.Verified {
		resp.ConfirmedAt = result.ConfirmedAt.UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) getReliability(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Bump == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "bump service is unavailable")
		return
	}
	rel, err := s.deps.Bump.Reliability(r.Context(), auth.User.ID)
	if err != nil {
		writeBumpError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, rel)
}

type publicReliabilityBandResponse struct {
	UserID string `json:"userId"`
	Band   string `json:"band"`
}

// getUserReliabilityBand is the public counterpart to getReliability
// (GET /v1/me/reliability): it lets any authenticated user look up another
// user's coarse, public-facing reliability Band — never the exact
// VerifiedBumpCount, which stays visible only to that user themselves via
// getReliability. See bump.Service.PublicBand's doc comment.
func (s *Server) getUserReliabilityBand(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Bump == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "bump service is unavailable")
		return
	}
	targetUserID := r.PathValue("userID")
	band, err := s.deps.Bump.PublicBand(r.Context(), auth.User.ID, targetUserID)
	if err != nil {
		writeBumpError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, publicReliabilityBandResponse{UserID: targetUserID, Band: string(band)})
}

type bumpVaultEntryResponse struct {
	SlotID        string `json:"slotId"`
	SlotTitle     string `json:"slotTitle"`
	CounterpartID string `json:"counterpartUserId"`
	ConfirmedAt   string `json:"confirmedAt"`
}

func (s *Server) getBumpVault(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Bump == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "bump service is unavailable")
		return
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil {
			limit = n
		}
	}
	entries, err := s.deps.Bump.Vault(r.Context(), auth.User.ID, limit)
	if err != nil {
		writeBumpError(w, r, err)
		return
	}
	out := make([]bumpVaultEntryResponse, 0, len(entries))
	for _, e := range entries {
		out = append(out, bumpVaultEntryResponse{
			SlotID:        e.SlotID,
			SlotTitle:     e.SlotTitle,
			CounterpartID: e.CounterpartID,
			ConfirmedAt:   e.ConfirmedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func writeBumpError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, bump.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_input", "invalid request")
	case errors.Is(err, bump.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "not_found", "slot not found")
	case errors.Is(err, bump.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "not authorized")
	case errors.Is(err, bump.ErrNotEligible):
		writeProblem(w, r, http.StatusConflict, "not_eligible", "not eligible to bump for this slot")
	case errors.Is(err, bump.ErrInvalidChallenge):
		writeProblem(w, r, http.StatusBadRequest, "invalid_challenge", "invalid or expired bump challenge")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
