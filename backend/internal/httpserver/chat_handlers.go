package httpserver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
)

type sendChatRequest struct {
	Text string `json:"text"`
}

type chatMessagesResponse struct {
	Items []chat.Message `json:"items"`
}

type splitBillRequest struct {
	TotalMinor     int      `json:"totalMinor"`
	ParticipantIDs []string `json:"participantIds"`
}

func (s *Server) sendChatMessage(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Chats == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "chat service is unavailable")
		return
	}
	var in sendChatRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	out, err := s.deps.Chats.Send(r.Context(), auth.User.ID, r.PathValue("slotID"), key, in.Text)
	if err != nil {
		s.writeChatError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

func (s *Server) listChatMessages(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Chats == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "chat service is unavailable")
		return
	}
	limit := chat.MaxRecent
	if raw := r.URL.Query().Get("limit"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 || parsed > chat.MaxRecent {
			writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid chat limit")
			return
		}
		limit = parsed
	}
	items, err := s.deps.Chats.ListRecent(r.Context(), auth.User.ID, r.PathValue("slotID"), limit)
	if err != nil {
		s.writeChatError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, chatMessagesResponse{Items: items})
}

// splitBill implements README §6.18's Bill Splitter calculator over HTTP.
// It is a stateless calculation, not persisted anywhere and never sent to
// any payment processor -- callers get the computed split back and decide
// what to do with it (share it in chat as a regular message, display it,
// etc).
func (s *Server) splitBill(w http.ResponseWriter, r *http.Request) {
	auth, _ := authFrom(r)
	if s.deps.Chats == nil {
		writeProblem(w, r, http.StatusServiceUnavailable, "not_ready", "chat service is unavailable")
		return
	}
	var in splitBillRequest
	if err := decodeJSON(w, r, &in); err != nil {
		writeProblem(w, r, http.StatusBadRequest, "invalid_request", "invalid JSON body")
		return
	}
	out, err := s.deps.Chats.SplitBill(r.Context(), auth.User.ID, r.PathValue("slotID"), in.TotalMinor, in.ParticipantIDs)
	if err != nil {
		s.writeChatError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) writeChatError(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, chat.ErrInvalidInput):
		writeProblem(w, r, http.StatusBadRequest, "invalid_chat", "invalid chat data")
	case errors.Is(err, chat.ErrNotFound):
		writeProblem(w, r, http.StatusNotFound, "slot_not_found", "slot not found")
	case errors.Is(err, chat.ErrForbidden):
		writeProblem(w, r, http.StatusForbidden, "chat_forbidden", "chat access is not allowed")
	case errors.Is(err, chat.ErrClosed):
		writeProblem(w, r, http.StatusConflict, "chat_closed", "chat is closed")
	case errors.Is(err, chat.ErrIdempotencyConflict):
		writeProblem(w, r, http.StatusConflict, "idempotency_conflict", "idempotency key was already used with different message data")
	default:
		writeProblem(w, r, http.StatusInternalServerError, "internal_error", "request failed")
	}
}
