package httpserver

import (
	"errors"
	"net/http"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/friend"
)

type friendRequestResponse struct {
	Outcome string `json:"outcome"`
}

type friendListResponse struct {
	Items []friend.Friend `json:"items"`
}

type friendPendingRequestListResponse struct {
	Items []friend.PendingRequest `json:"items"`
}

func (s *Server) sendFriendRequest(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	result, err := s.deps.Friends.Request(r.Context(), auth.User.ID, r.PathValue("userID"))
	if err != nil {
		writeFriendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, friendRequestResponse{Outcome: string(result.Outcome)})
}

func (s *Server) acceptFriendRequest(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	if err := s.deps.Friends.Accept(r.Context(), auth.User.ID, r.PathValue("userID")); err != nil {
		writeFriendError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) rejectFriendRequest(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	if err := s.deps.Friends.Reject(r.Context(), auth.User.ID, r.PathValue("userID")); err != nil {
		writeFriendError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) cancelFriendRequest(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	if err := s.deps.Friends.Cancel(r.Context(), auth.User.ID, r.PathValue("userID")); err != nil {
		writeFriendError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) removeFriend(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	if err := s.deps.Friends.Remove(r.Context(), auth.User.ID, r.PathValue("userID")); err != nil {
		writeFriendError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listFriends(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	items, err := s.deps.Friends.ListFriends(r.Context(), auth.User.ID)
	if err != nil {
		writeFriendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, friendListResponse{Items: items})
}

func (s *Server) listIncomingFriendRequests(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	items, err := s.deps.Friends.ListIncoming(r.Context(), auth.User.ID)
	if err != nil {
		writeFriendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, friendPendingRequestListResponse{Items: items})
}

func (s *Server) listOutgoingFriendRequests(w http.ResponseWriter, r *http.Request) {
	auth, ok := authFrom(r)
	if !ok {
		writeProblem(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return
	}
	if s.deps.Friends == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "friend service is unavailable")
		return
	}
	items, err := s.deps.Friends.ListOutgoing(r.Context(), auth.User.ID)
	if err != nil {
		writeFriendError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, friendPendingRequestListResponse{Items: items})
}

func writeFriendError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, friend.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_friend_request", "invalid friend request")
	case errors.Is(err, friend.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "forbidden", "operation is not allowed")
	case errors.Is(err, friend.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "friend_request_not_found", "friend request not found")
	case errors.Is(err, friend.ErrAlreadyFriends):
		writeProblem(w, r, http.StatusConflict, "already_friends", "users are already friends")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
