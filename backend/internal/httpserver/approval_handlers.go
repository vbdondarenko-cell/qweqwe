package httpserver

import (
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/push"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type pendingRequestsResponse struct {
	Items []slot.PendingRequest `json:"items"`
}

func (s *Server) requestSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	if !s.requireWaitlistCapabilityForSlot(w, r, auth.User.ID, r.PathValue("slotID")) {
		return
	}
	out, err := s.deps.Slots.Request(r.Context(), auth.User.ID, r.PathValue("slotID"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	if out.ViewerState == slot.ViewerPending {
		s.notifyUser(out.Organizer.ID, push.Message{
			Title: "New LinkUp request",
			Body:  auth.User.DisplayName + " wants to join " + out.Title,
			Data:  map[string]string{"type": "slot_request", "slotId": out.ID},
		})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) leaveSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	out, err := s.deps.Slots.Leave(r.Context(), auth.User.ID, r.PathValue("slotID"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) listPendingRequests(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	items, err := s.deps.Slots.ListPending(r.Context(), auth.User.ID, r.PathValue("slotID"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, pendingRequestsResponse{Items: items})
}

func (s *Server) approveRequest(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	requesterID := r.PathValue("userID")
	out, err := s.deps.Slots.Approve(r.Context(), auth.User.ID, r.PathValue("slotID"), requesterID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	s.notifyUser(requesterID, push.Message{
		Title: "Request approved",
		Body:  "You were approved for " + out.Title,
		Data:  map[string]string{"type": "slot_request_approved", "slotId": out.ID},
	})
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) rejectRequest(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	requesterID := r.PathValue("userID")
	out, err := s.deps.Slots.Reject(r.Context(), auth.User.ID, r.PathValue("slotID"), requesterID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	s.notifyUser(requesterID, push.Message{
		Title: "Request update",
		Body:  "Your request for " + out.Title + " was not approved",
		Data:  map[string]string{"type": "slot_request_rejected", "slotId": out.ID},
	})
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) startSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	out, err := s.deps.Slots.Start(r.Context(), auth.User.ID, r.PathValue("slotID"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) completeSlot(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Slots == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "slot service is unavailable")
		return
	}
	out, err := s.deps.Slots.Complete(r.Context(), auth.User.ID, r.PathValue("slotID"), r.Header.Get("Idempotency-Key"))
	if err != nil {
		s.writeSlotError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}
