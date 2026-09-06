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
}

func (s *chatHTTPStore) Send(_ context.Context, actorID, slotID, messageID, text string) (chat.Message, error) {
	if s.sendErr != nil { return chat.Message{}, s.sendErr }
	return chat.Message{
		ID: messageID,
		SlotID: slotID,
		Author: chat.Author{ID: actorID, Username: "alice", DisplayName: "Alice"},
		Text: text,
		CreatedAt: time.Unix(1, 0).UTC(),
	}, nil
}
func (s *chatHTTPStore) ListRecent(_ context.Context, _, _ string, _ int) ([]chat.Message, error) {
	if s.listErr != nil { return nil, s.listErr }
	return s.items, nil
}

func TestChatSendAndReadHTTP(t *testing.T) {
	accounts, err := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
	if err != nil { t.Fatal(err) }
	chatStore := &chatHTTPStore{}
	chats, err := chat.NewService(chatStore)
	if err != nil { t.Fatal(err) }
	server := New(Dependencies{Accounts: accounts, Chats: chats})
	token := registerHTTPUser(t, server)

	send := httptest.NewRequest(http.MethodPost, "/v1/slots/slot-1/chat/messages", bytes.NewBufferString(`{"text":" hello "}`))
	send.Header.Set("Authorization", "Bearer "+token)
	sendRec := httptest.NewRecorder(); server.Handler().ServeHTTP(sendRec, send)
	if sendRec.Code != http.StatusCreated { t.Fatalf("send status=%d body=%s", sendRec.Code, sendRec.Body.String()) }
	var sent chat.Message
	if err := json.NewDecoder(sendRec.Body).Decode(&sent); err != nil { t.Fatal(err) }
	if sent.Text != "hello" || sent.Author.Username != "alice" { t.Fatalf("unexpected sent message: %#v", sent) }

	chatStore.items = []chat.Message{sent}
	read := httptest.NewRequest(http.MethodGet, "/v1/slots/slot-1/chat/messages?limit=50", nil)
	read.Header.Set("Authorization", "Bearer "+token)
	readRec := httptest.NewRecorder(); server.Handler().ServeHTTP(readRec, read)
	if readRec.Code != http.StatusOK { t.Fatalf("read status=%d body=%s", readRec.Code, readRec.Body.String()) }
}

func TestChatForbiddenAndClosedAreExplicit(t *testing.T) {
	for _, tc := range []struct {
		name string
		err error
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
			rec := httptest.NewRecorder(); server.Handler().ServeHTTP(rec, req)
			if rec.Code != tc.want { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
		})
	}
}

func TestChatRejectsInvalidLimitAndOversizedMessage(t *testing.T) {
	accounts, _ := account.NewService(&authTestStore{}, password.OWASPMinimum(), time.Hour)
	chats, _ := chat.NewService(&chatHTTPStore{})
	server := New(Dependencies{Accounts: accounts, Chats: chats})
	token := registerHTTPUser(t, server)

	badLimit := httptest.NewRequest(http.MethodGet, "/v1/slots/slot-1/chat/messages?limit=101", nil)
	badLimit.Header.Set("Authorization", "Bearer "+token)
	badLimitRec := httptest.NewRecorder(); server.Handler().ServeHTTP(badLimitRec, badLimit)
	if badLimitRec.Code != http.StatusBadRequest { t.Fatalf("limit status=%d", badLimitRec.Code) }
}
