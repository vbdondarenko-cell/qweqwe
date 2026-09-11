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
	// README §6.8: "push is never called directly from an HTTP handler".
	// The host's "new request" notification now projects through the
	// notification outbox pipeline (NotificationProjector, slot.request_created)
	// instead of being sent synchronously from here.
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
	// README §6.8: "push is never called directly from an HTTP handler".
	// The requester's "you're in" notification now projects through the
	// notification outbox pipeline (NotificationProjector, slot.membership_added)
	// instead of being sent synchronously from here.
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
	// Known deviation from README §6.8's "push is never called directly
	// from an HTTP handler": the generic slot.request_removed outbox event
	// this mutation emits cannot distinguish a host rejection from the
	// requester's own withdrawal (slot_requests carries no "removed by"
	// actor for the trigger to record), so NotificationProjector cannot
	// build this specific notification from the outbox alone yet without
	// risking telling a user who withdrew their own request that it was
	// rejected. Left as a direct call — a wrong "not approved" message
	// would be worse than an un-pipelined correct one — until a distinct
	// rejection signal exists to build it from properly.
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
