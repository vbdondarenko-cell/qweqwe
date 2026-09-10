package httpserver

import (
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/capability"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

// requireWaitlistCapabilityForSlot is deliberately selective: APPROVAL/INSTANT
// keep their existing availability while WAITLIST remains server fail-closed.
func (s *Server) requireWaitlistCapabilityForSlot(w http.ResponseWriter, r *http.Request, userID, slotID string) bool {
	current, err := s.deps.Slots.Get(r.Context(), userID, slotID)
	if err != nil {
		s.writeSlotError(w, r, err)
		return false
	}
	if current.AccessMode != slot.AccessWaitlist {
		return true
	}
	if s.deps.Capabilities == nil || !s.deps.Capabilities.Enabled(r.Context(), userID, capability.Waitlist) {
		writeProblem(w, r, http.StatusForbidden, "capability_disabled", "capability is not active")
		return false
	}
	return true
}
