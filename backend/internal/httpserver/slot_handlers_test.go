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
	candidate.ViewerState = slot.ViewerHost
	s.current = candidate
	return candidate, nil
}
func (s *slotHTTPStore) Get(_ context.Context, _, slotID string) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) ListPulse(_ context.Context, _ string, _ int) ([]slot.Slot, error) {
	if s.current.ID == "" || s.current.State == slot.StateCancelled || s.current.State == slot.StateCompleted || s.current.State == slot.StateActive { return []slot.Slot{}, nil }
	return []slot.Slot{s.current}, nil
}
func (s *slotHTTPStore) Edit(_ context.Context, _, slotID string, patch slot.EditInput, _ string, _ []byte, now time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	if patch.ExpectedVersion != s.current.Version { return slot.Slot{}, slot.ErrConflict }
	if patch.Title != nil { s.current.Title = *patch.Title }
	s.current.Version++
	s.current.UpdatedAt = now
	return s.current, nil
}
func (s *slotHTTPStore) Cancel(_ context.Context, _, slotID string, expectedVersion int64, _ string, _ []byte, now time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	if expectedVersion != s.current.Version { return slot.Slot{}, slot.ErrConflict }
	s.current.State = slot.StateCancelled
	s.current.Version++
	s.current.UpdatedAt = now
	return s.current, nil
}
func (s *slotHTTPStore) Request(_ context.Context, _, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) Leave(_ context.Context, _, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) ListPending(_ context.Context, _, slotID string) ([]slot.PendingRequest, error) {
	if s.current.ID != slotID { return nil, slot.ErrNotFound }
	return []slot.PendingRequest{}, nil
}
func (s *slotHTTPStore) Approve(_ context.Context, _, slotID, _ string, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) Reject(_ context.Context, _, slotID, _ string, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) Start(_ context.Context, _, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}
func (s *slotHTTPStore) Complete(_ context.Context, _, slotID, _ string, _ []byte, _ time.Time) (slot.Slot, error) {
	if s.current.ID != slotID { return slot.Slot{}, slot.ErrNotFound }
	return s.current, nil
}

func TestSlotCreatePulseEditCancelHTTPFlow(t *testing.T) {
	accountStore := &authTestStore{}
	accounts, err := account.NewService(accountStore, password.OWASPMinimum(), time.Hour)
	if err != nil { t.Fatal(err) }
	slotStore := &slotHTTPStore{}
	slots, err := slot.NewService(slotStore)
	if err != nil { t.Fatal(err) }
	server := New(Dependencies{Accounts: accounts, Slots: slots})
	token := registerHTTPUser(t, server)

	create := httptest.NewRequest(http.MethodPost, "/v1/slots", bytes.NewBufferString(`{"title":"Coffee","activity":"coffee","placeText":"Podil","capacity":4}`))
	create.Header.Set("Authorization", "Bearer "+token)
	create.Header.Set("Idempotency-Key", "slot-create-http-001")
	createRec := httptest.NewRecorder(); server.Handler().ServeHTTP(createRec, create)
	if createRec.Code != http.StatusCreated { t.Fatalf("create status=%d body=%s", createRec.Code, createRec.Body.String()) }
	var created slot.Slot
	if err := json.NewDecoder(createRec.Body).Decode(&created); err != nil { t.Fatal(err) }
	if created.AccessMode != slot.AccessApproval || created.Visibility != slot.VisibilityPublic || created.ViewerState != slot.ViewerHost { t.Fatalf("unexpected create contract: %#v", created) }

	for _, tc := range []struct { path, bearer string; status, count int }{
        {"/v1/me/slots?view=HOSTING&actorId=another-user", token, http.StatusOK, 1},
        {"/v1/me/slots", token, http.StatusOK, 1},
        {"/v1/me/slots?view=JOINED", token, http.StatusOK, 0},
        {"/v1/me/slots?view=REQUESTED", token, http.StatusOK, 0},
        {"/v1/me/slots?view=ALL_USERS", token, http.StatusBadRequest, 0},
        {"/v1/me/slots", "", http.StatusUnauthorized, 0},
    } {
        req := httptest.NewRequest(http.MethodGet, tc.path, nil)
        if tc.bearer != "" { req.Header.Set("Authorization", "Bearer "+tc.bearer) }
        rec := httptest.NewRecorder()
        server.Handler().ServeHTTP(rec, req)
        if rec.Code != tc.status { t.Fatalf("%s status=%d body=%s", tc.path, rec.Code, rec.Body.String()) }
        if tc.status == http.StatusOK {
            var body pulseResponse
            if err := json.NewDecoder(rec.Body).Decode(&body); err != nil { t.Fatal(err) }
            if body.Items == nil || len(body.Items) != tc.count { t.Fatalf("%s items=%#v", tc.path, body.Items) }
        }
    }

    for _, tc := range []struct { path, bearer string; status int }{
        {"/v1/slots/"+created.ID+"/accepted", token, http.StatusOK},
        {"/v1/slots/"+created.ID+"/accepted", "", http.StatusUnauthorized},
        {"/v1/slots/00000000-0000-0000-0000-000000000000/accepted", token, http.StatusNotFound},
    } {
        req := httptest.NewRequest(http.MethodGet, tc.path, nil)
        if tc.bearer != "" { req.Header.Set("Authorization", "Bearer "+tc.bearer) }
        rec := httptest.NewRecorder()
        server.Handler().ServeHTTP(rec, req)
        if rec.Code != tc.status { t.Fatalf("%s status=%d", tc.path, rec.Code) }
        if tc.status == http.StatusOK && rec.Body.String() != "{\"items\":[]}\n" { t.Fatalf("unexpected roster: %s", rec.Body.String()) }
    }

	pulse := httptest.NewRequest(http.MethodGet, "/v1/pulse", nil)
	pulse.Header.Set("Authorization", "Bearer "+token)
	pulseRec := httptest.NewRecorder(); server.Handler().ServeHTTP(pulseRec, pulse)
	if pulseRec.Code != http.StatusOK { t.Fatalf("pulse status=%d body=%s", pulseRec.Code, pulseRec.Body.String()) }

	edit := httptest.NewRequest(http.MethodPatch, "/v1/slots/"+created.ID, bytes.NewBufferString(`{"expectedVersion":1,"title":"Updated Coffee"}`))
	edit.Header.Set("Authorization", "Bearer "+token)
	edit.Header.Set("Idempotency-Key", "slot-edit-http-0001")
	editRec := httptest.NewRecorder(); server.Handler().ServeHTTP(editRec, edit)
	if editRec.Code != http.StatusOK { t.Fatalf("edit status=%d body=%s", editRec.Code, editRec.Body.String()) }

	stale := httptest.NewRequest(http.MethodPatch, "/v1/slots/"+created.ID, bytes.NewBufferString(`{"expectedVersion":1,"title":"Stale"}`))
	stale.Header.Set("Authorization", "Bearer "+token)
	stale.Header.Set("Idempotency-Key", "slot-edit-http-0002")
	staleRec := httptest.NewRecorder(); server.Handler().ServeHTTP(staleRec, stale)
	if staleRec.Code != http.StatusConflict { t.Fatalf("stale edit status=%d body=%s", staleRec.Code, staleRec.Body.String()) }

	cancel := httptest.NewRequest(http.MethodPost, "/v1/slots/"+created.ID+"/cancel", bytes.NewBufferString(`{"expectedVersion":2}`))
	cancel.Header.Set("Authorization", "Bearer "+token)
	cancel.Header.Set("Idempotency-Key", "slot-cancel-http-001")
	cancelRec := httptest.NewRecorder(); server.Handler().ServeHTTP(cancelRec, cancel)
	if cancelRec.Code != http.StatusOK { t.Fatalf("cancel status=%d body=%s", cancelRec.Code, cancelRec.Body.String()) }
}

func TestSlotCreateRequiresIdempotencyKey(t *testing.T) {
	accountStore := &authTestStore{}
	accounts, _ := account.NewService(accountStore, password.OWASPMinimum(), time.Hour)
	slots, _ := slot.NewService(&slotHTTPStore{})
	server := New(Dependencies{Accounts: accounts, Slots: slots})
	token := registerHTTPUser(t, server)
	req := httptest.NewRequest(http.MethodPost, "/v1/slots", bytes.NewBufferString(`{"title":"Coffee","activity":"coffee","placeText":"Podil","capacity":4}`))
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder(); server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest { t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String()) }
}

func TestApprovalRoutesRequireAuthentication(t *testing.T) {
	slots, _ := slot.NewService(&slotHTTPStore{})
	server := New(Dependencies{Slots: slots})
	for _, path := range []string{
		"/v1/slots/slot-id/request",
		"/v1/slots/slot-id/leave",
		"/v1/slots/slot-id/requests/user-id/approve",
		"/v1/slots/slot-id/requests/user-id/reject",
		"/v1/slots/slot-id/start",
		"/v1/slots/slot-id/complete",
	} {
		rec := httptest.NewRecorder()
		server.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodPost, path, nil))
		if rec.Code != http.StatusServiceUnavailable && rec.Code != http.StatusUnauthorized {
			t.Fatalf("path=%s status=%d body=%s", path, rec.Code, rec.Body.String())
		}
	}
}

func registerHTTPUser(t *testing.T, server *Server) string {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/auth/register", bytes.NewBufferString(`{"email":"alice@example.com","username":"alice","displayName":"Alice","password":"correct horse battery staple","language":"uk"}`))
	rec := httptest.NewRecorder(); server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated { t.Fatalf("register status=%d body=%s", rec.Code, rec.Body.String()) }
	var auth account.AuthResult
	if err := json.NewDecoder(rec.Body).Decode(&auth); err != nil { t.Fatal(err) }
	return auth.Token
}

func (s *slotHTTPStore) ListMine(_ context.Context, actorID, view string, _ int) ([]slot.Slot, error) {
    if s.current.ID != "" && s.current.Organizer.ID == actorID && view == "HOSTING" { return []slot.Slot{s.current},nil }
    return []slot.Slot{},nil
}

func (s *slotHTTPStore) ListAccepted(_ context.Context, actorID, slotID string) ([]slot.Organizer, error) {
    if s.current.ID != slotID || s.current.Organizer.ID != actorID { return nil, slot.ErrNotFound }
    return []slot.Organizer{}, nil
}
