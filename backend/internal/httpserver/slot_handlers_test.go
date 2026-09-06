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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/password"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/slot"
)

type slotHTTPStore struct {
	current slot.Slot
}

func (s *slotHTTPStore) Create(_ context.Context, _ string, candidate slot.Slot, _ string, _ []byte) (slot.Slot, error) {
	candidate.Organizer.Username = "alice"
	candidate.Organizer.DisplayName = "Alice"
	s.current = candidate
	return candidate, nil
}

func (s *slotHTTPStore) Get(_ context.Context, _, slotID string) (slot.Slot, error) {
	if s.current.ID != slotID {
		return slot.Slot{}, slot.ErrNotFound
	}
	return s.current, nil
}

func (s *slotHTTPStore) ListPulse(_ context.Context, _ string, _ int) ([]slot.Slot, error) {
	if s.current.ID == "" || s.current.State == slot.StateCancelled {
		return []slot.Slot{}, nil
	}
	return []slot.Slot{s.current}, nil
}

func (s *slotHTTPStore) Edit(_ context.Context, _, slotID string, patch slot.EditInput, _ string, _ []byte, now time.Time) (slot.Slot, error) {
	if s.current.ID != slotID {
		return slot.Slot{}, slot.ErrNotFound
	}
	if patch.ExpectedVersion != s.current.Version {
		return slot.Slot{}, slot.ErrConflict
	}
	if patch.Title != nil {
		s.current.Title = *patch.Title
	}
	s.current.Version++
	s.current.UpdatedAt = now
	return s.current, nil
}

func (s *slotHTTPStore) Cancel(_ context.Context, _, slotID string, expectedVersion int64, _ string, _ []byte, now time.Time) (slot.Slot, error) {
	if s.current.ID != slotID {
		return slot.Slot{}, slot.ErrNotFound
	}
	if expectedVersion != s.current.Version {
		return slot.Slot{}, slot.ErrConflict
	}
	s.current.State = slot.StateCancelled
	s.current.Version++
	s.current.UpdatedAt = now
	return s.current, nil
}

func TestSlotCreatePulseEditCancelHTTPFlow(t *testing.T) {
	accountStore := &authTestStore{}
	accounts, err := account.NewService(accountStore, password.OWASPMinimum(), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	slotStore := &slotHTTPStore{}
	slots, err := slot.NewService(slotStore)
	if err != nil {
		t.Fatal(err)
	}
	server := New(Dependencies{Accounts: accounts, Slots: slots})

	token := registerHTTPUser(t, server)

	create := httptest.NewRequest(http.MethodPost, "/v1/slots", bytes.NewBufferString(`{"title":"Coffee","activity":"coffee","placeText":"Podil","capacity":4}`))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Idempotency-Key", "slot-create-http-001")
	createRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String())
	}
	var created slot.Slot
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.AccessMode != slot.AccessApproval || created.Visibility != slot.VisibilityPublic {
		t.Fatalf("unexpected create contract: %#v", created)
	}

	pulse := httptest.NewRequest(http.MethodGet, "/v1/pulse", nil)
	pulse.Header.Set("Authorization", "Bearer "+token)
	pulseRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(pulseRec, pulse)
	if pulseRec.Code != http.StatusOK {
		t.Fatalf("pulse status=%d body=%s", pulseRec.Code, pulseRec.Body.String())
	}

	edit := httptest.NewRequest(http.MethodPatch, "/v1/slots/"+created.ID, bytes.NewBufferString(`{"expectedVersion":1,"title":"Updated Coffee"}`))
	edit.Header.Set("Authorization", "Bearer "+token)
	edit.Header.Set("Idempotency-Key", "slot-edit-http-0001")
	editRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(editRec, edit)
	if editRec.Code != http.StatusOK {
		t.Fatalf("edit status=%d body=%s", editRec.Code, editRec.Body.String())
	}

	stale := httptest.NewRequest(http.MethodPatch, "/v1/slots/"+created.ID, bytes.NewBufferString(`{"expectedVersion":1,"title":"Stale"}`))
	stale.Header.Set("Authorization", "Bearer "+token)
	stale.Header.Set("Idempotency-Key", "slot-edit-http-0002")
	staleRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(staleRec, stale)
	if staleRec.Code != http.StatusConflict {
		t.Fatalf("stale edit status=%d body=%s", staleRec.Code, staleRec.Body.String())
	}

	cancel := httptest.NewRequest(http.MethodPost, "/v1/slots/"+created.ID+"/cancel", bytes.NewBufferString(`{"expectedVersion":2}`))
	cancel.Header.Set("Authorization", "Bearer "+token)
	cancel.Header.Set("Idempotency-Key", "slot-cancel-http-001")
	cancelRec := httptest.NewRecorder()
	server.Handler().ServeHTTP(cancelRec, cancel)
	if cancelRec.Code != http.StatusOK {
		t.Fatalf("cancel status=%d body=%s", cancelRec.Code, cancelRec.Body.String())
	}
}

func TestSlotCreateRequiresIdempotencyKey(t *testing.T) {
	accountStore := &authTestStore{}
	accounts, _ := account.NewService(accountStore, password.OWASPMinimum(), time.Hour)
	slots, _ := slot.NewService(&slotHTTPStore{})
	server := New(Dependencies{Accounts: accounts, Slots: slots})
	token := registerHTTPUser(t, server)

	req := httptest.NewRequest(http.MethodPost, "/v1/slots", bytes.NewBufferString(`{"title":"Coffee","activity":"coffee","placeText":"Podil","capacity":4}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
}

func registerHTTPUser(t *testing.T, server *Server) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"email":"alice@example.com","username":"alice","displayName":"Alice","password":"correct horse battery staple","language":"uk"}`))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String())
	}
	var auth account.AuthResult
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil {
		t.Fatal(err)
	}
	return auth.Token
}
