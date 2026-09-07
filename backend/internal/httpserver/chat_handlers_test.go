package httpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vbdondarenko-cell/qweqwe/backend/internal/account"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/chat"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
)

type chatHTTPStore struct {
	sendErr error
	listErr error
	items   []chat.Message
	byKey   map[string]chat.Message
}

func (s *chatHTTPStore) Send(_ context.Context, actorID, slotID, messageID, key, text string) (chat.Message, error) {
	if s.sendErr != nil {
		return chat.Message{}, s.sendErr
	}
	if s.byKey == nil {
		s.byKey = make(map[string]chat.Message)
	}
	if existing, ok := s.byKey[key]; ok {
		if existing.Text != text {
			return chat.Message{}, chat.ErrIdempotencyConflict
		}
		return existing, nil
	}
	out := chat.Message{
		ID:        messageID,
		SlotID:    slotID,
		Author:    chat.Author{ID: actorID, Username: "alice", DisplayName: "Alice"},
		Text:      text,
		CreatedAt: time.Unix(1, 0).UTC(),
	}
	s.byKey[key] = out
	return out, nil
}

func (s *chatHTTPStore) ListRecent(_ context.Context, _, _ string, _ int) ([]chat.Message, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	return s.items, nil
}

func TestChatSendAndReadHTTP(t *testing.T) {
	accounts, err := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	chatStore := &chatHTTPStore{}
	chats, err := chat.NewService(chatStore)
	if err != nil {
		t.Fatal(err)
	}
	server := New(Dependencies{Accounts: accounts, Chats: chats})
	token := registerHTTPUser(t, server)
	key := "12345678-1234-4234-8234-123456789abc"

	sendMessage := func(text string) *httptest.ResponseRecorder {
		t.Helper()
		send := httptest.NewRequest(http.MethodPost, "/v1/slots/slot-1/chat/messages", bytes.NewBufferString(`{"text":"`+text+`"}`))
		send.Header.Set("Authorization", "Bearer "+token)
		send.Header.Set("Idempotency-Key", key)
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, send)
		return rec
	}

	firstRec := sendMessage(" hello ")
	if firstRec.Code != http.StatusCreated {
		t.Fatalf("send status=%d body=%s", firstRec.Code, firstRec.Body.String())
	}
	firstBody := append([]byte(nil), firstRec.Body.Bytes()...)
	var first chat.Message
	if err := json.Unmarshal(firstBody, &first); err != nil {
		t.Fatal(err)
	}
	if first.Text != "hello" || first.Author.Username != "alice" {
		t.Fatalf("unexpected sent message: %#v", first)
	}
	var raw map[string]any
	if err := json.Unmarshal(firstBody, &raw); err != nil {
		t.Fatal(err)
	}
	if _, leaked := raw["idempotencyKey"]; leaked {
		t.Fatal("chat idempotency key leaked in HTTP response")
	}

	replayRec := sendMessage("hello")
	if replayRec.Code != http.StatusCreated {
		t.Fatalf("replay status=%d body=%s", replayRec.Code, replayRec.Body.String())
	}
	var replay chat.Message
	if err := json.NewDecoder(replayRec.Body).Decode(&replay); err != nil {
		t.Fatal(err)
	}
	if replay.ID != first.ID {
		t.Fatalf("replay created duplicate message: first=%s replay=%s", first.ID, replay.ID)
	}

	conflictRec := sendMessage("different")
	if conflictRec.Code != http.StatusConflict {
		t.Fatalf("conflict status=%d body=%s", conflictRec.Code, conflictRec.Body.String())
	}

	chatStore.items = []chat.Message{first}
	read := httptest.NewRequest(http.MethodGet, "/v1/slots/slot-1/chat/messages?limit=50", nil)
	read.Header.Set("Authorization", "Bearer "+token)
	readRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(readRec, read)
	if readRec.Code != http.StatusOK {
		t.Fatalf("read status=%d body=%s", readRec.Code, readRec.Body.String())
	}
}

func TestChatSendRequiresIdempotencyKey(t *testing.T) {
	accounts, _ := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
	chats, _ := chat.NewService(&chatHTTPStore{})
	server := New(Dependencies{Accounts: accounts, Chats: chats})
	token := registerHTTPUser(t, server)
	req := httptest.NewRequest(http.MethodPost, "/v1/slots/slot-1/chat/messages", bytes.NewBufferString(`{"text":"hello"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func TestChatForbiddenAndClosedAreExplicit(t *testing.T) {
	for _, tc := range []struct {
		name string
		err  error
		want int
	}{
		{"forbidden", chat.ErrForbidden, http.StatusForbidden},
		{"closed", chat.ErrClosed, http.StatusConflict},
	} {
		t.Run(tc.name, func(t *testing.T) {
			accounts, _ := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
			store := &chatHTTPStore{listErr: tc.err}
			chats, _ := chat.NewService(store)
			server := New(Dependencies{Accounts: accounts, Chats: chats})
			token := registerHTTPUser(t, server)
			req := httptest.NewRequest(http.MethodGet, "/v1/slots/slot-1/chat/messages", nil)
			req.Header.Set("Authorization", "Bearer "+token)
			rec := httptest.NewRecorder()
			server.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
			}
		})
	}
}

func TestChatListLimitValidation(t *testing.T) {
	accounts, _ := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
	chats, _ := chat.NewService(&chatHTTPStore{})
	server := New(Dependencies{Accounts: accounts, Chats: chats})
	token := registerHTTPUser(t, server)
	req := httptest.NewRequest(http.MethodGet, "/v1/slots/slot-1/chat/messages?limit=0", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}
