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
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/guardian"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/session"
)

type fakeGuardianStore struct {
	createErr    error
	revokeErr    error
	accessStatus guardian.Status
	accessErr    error
}

func (f *fakeGuardianStore) CreateLink(context.Context, string, string, string, guardian.Mode, []byte, time.Time, time.Time) error {
	return f.createErr
}
func (f *fakeGuardianStore) Revoke(context.Context, string, string, time.Time) error {
	return f.revokeErr
}
func (f *fakeGuardianStore) Access(context.Context, []byte, time.Time) (guardian.Status, error) {
	if f.accessErr != nil {
		return guardian.Status{}, f.accessErr
	}
	return f.accessStatus, nil
}

func requestWithGuardianAuth(method, target, body string) *http.Request {
	var reader *bytes.Reader
	if body != "" {
		reader = bytes.NewReader([]byte(body))
	} else {
		reader = bytes.NewReader(nil)
	}
	request := httptest.NewRequest(method, target, reader)
	ctx := context.WithValue(request.Context(), authKey, authContext{User: account.User{ID: "11111111-1111-1111-1111-111111111111"}})
	return request.WithContext(ctx)
}

func TestCreateGuardianLinkReturnsTokenOnce(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	recorder := httptest.NewRecorder()
	request := requestWithGuardianAuth(http.MethodPost, "/v1/slots/slot-1/guardian-links", `{"mode":"STATUS_ONLY","ttlSeconds":3600}`)
	request.SetPathValue("slotID", "slot-1")
	server.createGuardianLink(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got guardianLinkResponse
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.Token == "" || got.Mode != "STATUS_ONLY" {
		t.Fatalf("unexpected response: %#v", got)
	}
}

func TestCreateGuardianLinkRejectsUnsupportedMode(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	recorder := httptest.NewRecorder()
	request := requestWithGuardianAuth(http.MethodPost, "/v1/slots/slot-1/guardian-links", `{"mode":"LIVE_PRECISE","ttlSeconds":3600}`)
	request.SetPathValue("slotID", "slot-1")
	server.createGuardianLink(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var problem problem
	if err := json.NewDecoder(recorder.Body).Decode(&problem); err != nil {
		t.Fatal(err)
	}
	if problem.Code != "mode_not_supported" {
		t.Fatalf("expected mode_not_supported, got %q", problem.Code)
	}
}

func TestCreateGuardianLinkFailsClosedWhenServiceUnavailable(t *testing.T) {
	server := &Server{}
	recorder := httptest.NewRecorder()
	server.createGuardianLink(recorder, requestWithGuardianAuth(http.MethodPost, "/v1/slots/slot-1/guardian-links", `{"mode":"STATUS_ONLY","ttlSeconds":3600}`))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestCreateGuardianLinkRequiresAuth(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/slots/slot-1/guardian-links", bytes.NewReader([]byte(`{"mode":"STATUS_ONLY","ttlSeconds":3600}`)))
	server.createGuardianLink(recorder, request)
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestRevokeGuardianLinkMapsForbidden(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{revokeErr: guardian.ErrForbidden})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	recorder := httptest.NewRecorder()
	request := requestWithGuardianAuth(http.MethodDelete, "/v1/slots/slot-1/guardian-links/link-1", "")
	request.SetPathValue("linkID", "link-1")
	server.revokeGuardianLink(recorder, request)
	if recorder.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestAccessGuardianLinkIsUnauthenticatedAndReturnsStatus(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{accessStatus: guardian.Status{SharerDisplayName: "Alice", SlotStatus: guardian.SlotStatusActive}})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	rawToken, _, err := session.Generate()
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/guardian-links/"+rawToken, nil)
	request.SetPathValue("token", rawToken)
	server.accessGuardianLink(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var got guardian.Status
	if err := json.NewDecoder(recorder.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.SharerDisplayName != "Alice" || got.SlotStatus != guardian.SlotStatusActive {
		t.Fatalf("unexpected status: %#v", got)
	}
}

func TestAccessGuardianLinkMapsUnavailableUniformly(t *testing.T) {
	svc, err := guardian.NewService(&fakeGuardianStore{accessErr: guardian.ErrLinkUnavailable})
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{deps: Dependencies{Guardians: svc}}
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/guardian-links/whatever", nil)
	request.SetPathValue("token", "whatever")
	server.accessGuardianLink(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}
